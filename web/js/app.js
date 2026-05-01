const ws = new WSClient();
const charts = {};
let lapsData = [];
let selectedLapIndex = 0;
let pinnedLiveLap = true;
let liveLap = null;
let currentLiveLapNum = -1;
let lastChartUpdate = 0;
const CHART_UPDATE_INTERVAL = 200;
let currentVehicleModel = '';
let gamePaused = false;
let circuitLength = 0; // meters, 0 = unknown

function initChart(id) {
  const el = document.getElementById(id);
  if (!el) return null;
  const chart = echarts.init(el, 'dark');
  charts[id] = chart;
  return chart;
}

function resizeAll() {
  Object.values(charts).forEach(c => c && c.resize());
}

// Tab switching
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
    setTimeout(resizeAll, 100);
  });
});

window.addEventListener('resize', resizeAll);

// Live lap accumulation
function resetLiveLap(snap) {
  const throttle = snap.throttle || 0;
  const brake = snap.brake || 0;
  const yaw = snap.rotation_yaw || 0;
  liveLap = {
    data_speed: [snap.speed || 0],
    data_throttle: [throttle],
    data_braking: [brake],
    data_coasting: [throttle === 0 && brake === 0 ? 1 : 0],
    data_rpm: [snap.rpm || 0],
    data_gear: [snap.gear || 0],
    data_boost: [snap.boost || 0],
    data_rotation_yaw: [yaw],
    data_absolute_yaw_rate_per_second: [0],
    data_position_x: [snap.position_x || 0],
    data_position_y: [snap.position_y || 0],
    data_position_z: [snap.position_z || 0],
    fuel_at_start: snap.fuel || 0,
    fuel_at_end: snap.fuel || 0,
    fuel_consumed: 0,
    lap_finish_time: 0,
    _is_live: true,
    _lap_ticks: 1,
    _yaw_history: [yaw],
    throttle_and_brake_ticks: throttle > 0 && brake > 0 ? 1 : 0,
    no_throttle_and_no_brake_ticks: throttle === 0 && brake === 0 ? 1 : 0,
    full_brake_ticks: brake >= 100 ? 1 : 0,
    full_throttle_ticks: throttle >= 100 ? 1 : 0,
  };
}

function accumulateLiveLap(snap) {
  if (!liveLap) return;
  const throttle = snap.throttle || 0;
  const brake = snap.brake || 0;
  const yaw = snap.rotation_yaw || 0;

  liveLap.data_speed.push(snap.speed || 0);
  liveLap.data_throttle.push(throttle);
  liveLap.data_braking.push(brake);
  liveLap.data_coasting.push(throttle === 0 && brake === 0 ? 1 : 0);
  liveLap.data_rpm.push(snap.rpm || 0);
  liveLap.data_gear.push(snap.gear || 0);
  liveLap.data_boost.push(snap.boost || 0);
  liveLap.data_rotation_yaw.push(yaw);
  liveLap._yaw_history.push(yaw);
  if (liveLap._yaw_history.length > 60) {
    liveLap.data_absolute_yaw_rate_per_second.push(
      Math.abs(yaw - liveLap._yaw_history[liveLap._yaw_history.length - 61])
    );
  } else {
    liveLap.data_absolute_yaw_rate_per_second.push(0);
  }
  liveLap.data_position_x.push(snap.position_x || 0);
  liveLap.data_position_y.push(snap.position_y || 0);
  liveLap.data_position_z.push(snap.position_z || 0);
  liveLap.fuel_at_end = snap.fuel || 0;
  liveLap._lap_ticks++;

  if (throttle > 0 && brake > 0) liveLap.throttle_and_brake_ticks++;
  if (throttle === 0 && brake === 0) liveLap.no_throttle_and_no_brake_ticks++;
  if (brake >= 100) liveLap.full_brake_ticks++;
  if (throttle >= 100) liveLap.full_throttle_ticks++;

  liveLap.fuel_consumed = liveLap.fuel_at_start - liveLap.fuel_at_end;
}

function getLapsForCharts() {
  if (!liveLap) return lapsData;
  return lapsData.concat(liveLap);
}

// WebSocket handlers
ws.on('telemetry', (data) => {
  const snap = data.data || data;
  gamePaused = snap.is_paused || false;

  if (!snap.in_race) {
    liveLap = null;
    currentLiveLapNum = -1;
    document.getElementById('lap-info').textContent = '';
    return;
  }

  currentVehicleModel = snap.vehicle_model || '';
  if (snap.circuit_length) circuitLength = snap.circuit_length;
  const pauseText = gamePaused ? ' [PAUSED]' : '';
  document.getElementById('lap-info').textContent =
    `Lap ${snap.current_lap}/${snap.total_laps}  ${currentVehicleModel}${pauseText}`;

  // Freeze charts during pause — no data accumulation, no chart updates
  if (gamePaused) return;

  if (snap.current_lap !== currentLiveLapNum) {
    liveLap = null;
    currentLiveLapNum = snap.current_lap;
  }

  if (!liveLap) {
    resetLiveLap(snap);
  } else {
    accumulateLiveLap(snap);
  }

  const now = Date.now();
  if (now - lastChartUpdate > CHART_UPDATE_INTERVAL) {
    lastChartUpdate = now;
    updateAllCharts();
  }
});

ws.on('lap_completed', () => {
  liveLap = null;
});

ws.on('laps_updated', (data) => {
  lapsData = data.laps || [];
  if (pinnedLiveLap) {
    selectedLapIndex = lapsData.length;
  }
  updateAllCharts();
});

ws.on('current_lap', (data) => {
  liveLap = data;
  currentLiveLapNum = data.number;
  currentVehicleModel = data.car_name || '';
  document.getElementById('lap-info').textContent =
    `Lap ${data.number}/${data.total_laps || '?'}  ${currentVehicleModel}`;
  updateAllCharts();
});

ws.on('disconnected', () => {
  document.getElementById('lap-info').textContent = '';
});

// Chart registry
const chartModules = {};

function registerChart(name, module) {
  chartModules[name] = module;
}

function initCharts() {
  Object.entries(chartModules).forEach(([name, mod]) => {
    const el = document.getElementById('chart-' + name);
    if (el) mod.init(el, charts);
  });
}

// Charts that show individual lap data work with live lap included;
// multi-lap analysis charts only get completed laps
const singleLapChartNames = ['speed', 'throttle', 'braking', 'coasting', 'yawrate', 'gear', 'rpm', 'boost', 'raceline', 'fuel', 'tires'];

const distanceChartNames = ['speed', 'throttle', 'braking', 'coasting', 'yawrate', 'gear', 'rpm', 'boost', 'tires', 'timediff', 'variance'];

function updateAllCharts() {
  const lapsWithLive = getLapsForCharts();
  const best = getBestLap(lapsWithLive);
  // Use the best lap's total distance as the x-axis max so all laps
  // render on the same scale. When the best lap is incomplete, fall
  // back to circuit length for proper right-aligned comparison.
  let xMax = null;
  if (best && best.data_speed) {
    const dist = xAxis(best.data_speed);
    xMax = dist[dist.length - 1];
  } else if (circuitLength > 0) {
    xMax = circuitLength / 1000;
  }
  Object.entries(chartModules).forEach(([name, mod]) => {
    if (!mod.update) return;
    if (pinnedLiveLap && singleLapChartNames.includes(name)) {
      mod.update(lapsWithLive, lapsWithLive.length - 1);
    } else {
      mod.update(lapsWithLive, selectedLapIndex);
    }
  });
  // Always set xAxis.max — when the best lap transitions from incomplete
  // to complete, the old max stays cached in ECharts if we skip the call.
  // null tells ECharts to auto-scale; circuit length pads incomplete laps.
  const max = xMax !== null ? xMax : null;
  distanceChartNames.forEach(name => {
    const chart = charts[name];
    if (chart) chart.setOption({ xAxis: { max } });
  });
  renderLapTable();
}

function selectLap(index) {
  selectedLapIndex = index;
  pinnedLiveLap = false;
  updateAllCharts();
}

function selectLiveLap() {
  pinnedLiveLap = true;
  updateAllCharts();
}

// Lap table
function renderLapTable() {
  const container = document.getElementById('lap-table');
  if (lapsData.length === 0 && !liveLap) {
    container.innerHTML = '';
    return;
  }

  const validLaps = lapsData.filter(l => l.lap_finish_time > 0 && !l.is_pit_lap);
  let bestTime = 0, worstTime = 0;
  if (validLaps.length > 0) {
    bestTime = validLaps.reduce((a, l) => Math.min(a, l.lap_finish_time), Infinity);
    worstTime = validLaps.reduce((a, l) => Math.max(a, l.lap_finish_time), 0);
  }

  let html = '<table><thead><tr>' +
    '<th>' + i18n.t('table.num') + '</th><th>' + i18n.t('table.time') + '</th><th>' + i18n.t('table.diff') + '</th><th>' + i18n.t('table.fuel') + '</th>' +
    '<th>' + i18n.t('table.thr') + '</th><th>' + i18n.t('table.brk') + '</th><th>' + i18n.t('table.cst') + '</th>' +
    '<th>' + i18n.t('table.spin') + '</th><th>' + i18n.t('table.tb') + '</th><th>' + i18n.t('table.car') + '</th>' +
    '</tr></thead><tbody>';

  const sortedRows = lapsData.map((l, i) => ({ lap: l, idx: i }))
    .sort((a, b) => (a.lap.number || a.idx + 1) - (b.lap.number || b.idx + 1));
  sortedRows.forEach(({ lap: l, idx: i }) => {
    const diff = l.lap_finish_time - (bestTime || 0);
    const diffStr = diff === 0 ? '--' : (diff > 0 ? '+' + msToTime(diff) : '-' + msToTime(-diff));
    let cls = '';
    if (!l.is_pit_lap && l.lap_finish_time > 0) {
      if (l.lap_finish_time === bestTime) cls = 'best';
      else if (l.lap_finish_time === worstTime) cls = 'worst';
    }
    if (i === selectedLapIndex) cls += (cls ? ' ' : '') + 'selected';
    html += '<tr class="' + cls + '" onclick="selectLap(' + i + ')">';
    html += `<td>${l.number || i + 1}${l.is_pit_lap ? ' <span class="pit-badge">PIT</span>' : ''}</td>`;
    html += `<td>${msToTime(l.lap_finish_time)}</td>`;
    html += `<td>${diffStr}</td>`;
    html += `<td>${l.fuel_consumed != null ? l.fuel_consumed : '-'}</td>`;
    html += `<td>${l.full_throttle_ticks != null ? Math.round(l.full_throttle_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.full_brake_ticks != null ? Math.round(l.full_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.no_throttle_and_no_brake_ticks != null ? Math.round(l.no_throttle_and_no_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.tires_spinning_ticks != null ? Math.round(l.tires_spinning_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.throttle_and_brake_ticks != null ? Math.round(l.throttle_and_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.car_name || '-'}</td>`;
    html += '</tr>';
  });

  // Live lap row
  if (liveLap) {
    const isLiveSelected = pinnedLiveLap;
    html += `<tr class="live${isLiveSelected ? ' selected' : ''}" onclick="selectLiveLap()">`;
    html += '<td>' + i18n.t('status.live') + '</td>';
    const lapSecs = liveLap._lap_ticks / 60;
    html += `<td>${Math.floor(lapSecs / 60)}:${(lapSecs % 60).toFixed(1).padStart(4, '0')}</td>`;
    html += '<td>--</td>';
    html += `<td>${liveLap.fuel_consumed != null ? liveLap.fuel_consumed : '-'}</td>`;
    const totalTicks = liveLap._lap_ticks || 1;
    html += `<td>${Math.round((liveLap.full_throttle_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.full_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.no_throttle_and_no_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.tires_spinning_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.throttle_and_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${currentVehicleModel || '-'}</td>`;
    html += '</tr>';
  }

  html += '</tbody></table>';
  container.innerHTML = html;
}

function getBestLap(laps) {
  let best = null;
  let bestTime = Infinity;
  for (const l of laps) {
    if (l._is_live || l.is_pit_lap || !l.lap_finish_time || l.lap_finish_time <= 0) continue;
    if (l.lap_finish_time < bestTime) { bestTime = l.lap_finish_time; best = l; }
  }
  return best;
}

function msToTime(ms) {
  const secs = ms / 1000;
  const mins = Math.floor(secs / 60);
  const remain = secs - mins * 60;
  return `${mins}:${remain.toFixed(3).padStart(6, '0')}`;
}

// Init on load
document.addEventListener('DOMContentLoaded', () => {
  i18n.apply();
  const targetLang = i18n.lang === 'zh' ? 'en' : 'zh';
  document.getElementById('lang-toggle').textContent = i18n.t('lang.' + targetLang);
  initCharts();
});
