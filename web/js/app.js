const ws = new WSClient();
const charts = {};
const SELECTED_TARGET_STORAGE_KEY = 'gt7_selected_target';
let lapsData = [];
let selectedTarget = loadSelectedTarget();
let liveLap = null;
let currentLiveLapNum = -1;
let lastTelemetrySequenceID = 0;
let lastChartUpdate = 0;
const CHART_UPDATE_INTERVAL = 200;
let currentVehicleModel = '';
let gamePaused = false;
let isReplayMode = false;
let isRaceComplete = false;
let lastPS5Connected = null;
let circuitLength = 0; // meters, 0 = unknown
let latestTelemetrySnapshot = null;

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
    updateAllCharts();
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
    data_tires: [snap.tire_slip || 0],
    data_boost: [snap.boost || 0],
    data_rotation_yaw: [yaw],
    data_absolute_yaw_rate_per_second: [0],
    data_position_x: [snap.position_x || 0],
    data_position_y: [snap.position_y || 0],
    data_position_z: [snap.position_z || 0],
    total_distance: 0,
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
  // Guard against current_lap message setting null data arrays
  if (!liveLap.data_speed) { resetLiveLap(snap); return; }
  const throttle = snap.throttle || 0;
  const brake = snap.brake || 0;
  const yaw = snap.rotation_yaw || 0;

  liveLap.data_speed.push(snap.speed || 0);
  liveLap.data_throttle.push(throttle);
  liveLap.data_braking.push(brake);
  liveLap.data_coasting.push(throttle === 0 && brake === 0 ? 1 : 0);
  liveLap.data_rpm.push(snap.rpm || 0);
  liveLap.data_gear.push(snap.gear || 0);
  liveLap.data_tires.push(snap.tire_slip || 0);
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
  // Accumulate total distance from 2D position deltas
  const px = liveLap.data_position_x;
  const pz = liveLap.data_position_z;
  const n = px.length;
  if (n >= 2) {
    const dx = px[n - 1] - px[n - 2];
    const dz = pz[n - 1] - pz[n - 2];
    liveLap.total_distance += Math.sqrt(dx * dx + dz * dz);
  }
  liveLap.fuel_at_end = snap.fuel || 0;
  liveLap._lap_ticks++;

  if (throttle > 0 && brake > 0) liveLap.throttle_and_brake_ticks++;
  if (throttle === 0 && brake === 0) liveLap.no_throttle_and_no_brake_ticks++;
  if (brake >= 100) liveLap.full_brake_ticks++;
  if (throttle >= 100) liveLap.full_throttle_ticks++;

  liveLap.fuel_consumed = liveLap.fuel_at_start - liveLap.fuel_at_end;
}

function getSelectableLaps() {
  if (!liveLap) return lapsData;
  return lapsData.concat(liveLap);
}

// WebSocket handlers
ws.on('telemetry', (data) => {
  const snap = data.data || data;
  latestTelemetrySnapshot = snap;
  gamePaused = snap.is_paused || false;
  isReplayMode = snap.is_replay || false;
  isRaceComplete = snap.is_race_complete || false;
  updatePS5Status(snap.ps5_connected);
  if (snap.sequence_id && snap.sequence_id === lastTelemetrySequenceID) {
    return;
  }
  lastTelemetrySequenceID = snap.sequence_id || lastTelemetrySequenceID;

  if (!snap.in_race) {
    document.getElementById('lap-info').textContent = '';
    renderDriverDashboard();
    return;
  }

  currentVehicleModel = snap.vehicle_model || '';
  if (snap.circuit_length) circuitLength = snap.circuit_length;
  const pauseText = gamePaused ? ' [PAUSED]' : '';
  const replayText = isReplayMode ? ' [REPLAY]' : '';
  const finishText = isRaceComplete ? ' [' + i18n.t('status.finished') + ']' : '';
  document.getElementById('lap-info').textContent =
    `Lap ${snap.current_lap}/${snap.total_laps}  ${currentVehicleModel}${pauseText}${replayText}${finishText}`;

  if (gamePaused || isRaceComplete) return;

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

ws.on('live_lap_diff', (data) => {
  if (liveLap && data.time_diff) {
    liveLap.time_diff = data.time_diff;
  }
});

ws.on('lap_completed', (data) => {
  const wasLiveSelected = isLiveLapSelected();
  const completedLap = data && (data.lap || data.data || data);

  liveLap = null;
  if (wasLiveSelected && completedLap) {
    selectedTarget = newHistoryTarget(0, completedLap);
  } else {
    normalizeSelectedTarget();
  }
  updateAllCharts();
});

ws.on('laps_updated', (data) => {
  lapsData = data.laps || [];
  for (const lap of lapsData) {
    if (lap.data_speed && lap.lap_ticks > 0 && lap.lap_finish_time > 0) {
      lap.data_speed._tickInterval = (lap.lap_finish_time / 1000) / lap.lap_ticks;
    }
  }
  normalizeSelectedTarget();
  updateAllCharts();
  refreshLapFiles();
});

ws.on('current_lap', (data) => {
  liveLap = data;
  currentLiveLapNum = data.number;
  currentVehicleModel = data.car_name || '';
  if (!liveLap.data_tires) {
    liveLap.data_tires = [];
  }
  if (liveLap.data_speed) {
    liveLap.data_speed._tickInterval = 1 / 60;
  }
  document.getElementById('lap-info').textContent =
    `Lap ${data.number}/${data.total_laps || '?'}  ${currentVehicleModel}`;
  updateAllCharts();
});

ws.on('current_lap_cleared', () => {
  liveLap = null;
  currentLiveLapNum = -1;
  normalizeSelectedTarget();
  updateAllCharts();
});

ws.on('telemetry_status', (data) => {
  updatePS5Status(data.ps5_connected);
});

ws.on('disconnected', () => {
  document.getElementById('lap-info').textContent = '';
  latestTelemetrySnapshot = null;
  updatePS5Status(false, 0);
  renderDriverDashboard();
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

const distanceChartNames = ['speed', 'throttle', 'braking', 'coasting', 'yawrate', 'gear', 'rpm', 'boost', 'tires', 'variance', 'timediff'];

function updateAllCharts() {
  const selectableLaps = getSelectableLaps();
  const targetLapIndex = getSelectedLapIndex(selectableLaps);
  const best = getBestLap(selectableLaps);
  const visibleChartNames = getVisibleChartNames();
  // Use the best lap's total distance as the x-axis max so all laps
  // render on the same scale. When the best lap is incomplete, fall
  // back to circuit length for proper right-aligned comparison.
  let xMax = null;
  if (best && best.total_distance) {
    xMax = best.total_distance / 1000;
  } else if (best && best.data_speed) {
    const dist = xAxis(best.data_speed);
    xMax = dist[dist.length - 1];
  } else if (circuitLength > 0) {
    xMax = circuitLength / 1000;
  }
  Object.entries(chartModules).forEach(([name, mod]) => {
    if (!mod.update) return;
    if (!visibleChartNames.has(name)) return;
    mod.update(selectableLaps, targetLapIndex);
  });
  // Always set xAxis.max — when the best lap transitions from incomplete
  // to complete, the old max stays cached in ECharts if we skip the call.
  // null tells ECharts to auto-scale; circuit length pads incomplete laps.
  const max = xMax !== null ? xMax : null;
  const axisLabel = {
    formatter: (v) => {
      const vm = Math.round(v * 1000);
      if (max != null && Math.abs(v - max) < 0.0005) {
        return vm + 'm';
      }
      return vm.toString();
    },
    fontSize: 9,
  };
  distanceChartNames.forEach(name => {
    if (!visibleChartNames.has(name)) return;
    const chart = charts[name];
    if (chart) chart.setOption({ xAxis: { max, axisLabel }, grid: { right: 25 } });
  });
  renderLapTable();
  renderDriverDashboard();
  if (visibleChartNames.has('fuel')) {
    renderShiftAnalysis(selectableLaps, targetLapIndex);
  }
}

function getVisibleChartNames() {
  const activePane = document.querySelector('.tab-pane.active');
  const names = new Set();
  if (!activePane) return names;

  activePane.querySelectorAll('[id^="chart-"]').forEach(el => {
    names.add(el.id.replace('chart-', ''));
  });
  return names;
}

function selectLap(index) {
  selectedTarget = newHistoryTarget(index);
  saveSelectedTarget();
  updateAllCharts();
}

function selectLiveLap() {
  selectedTarget = newLiveTarget();
  saveSelectedTarget();
  updateAllCharts();
}

function newLiveTarget() {
  return { type: 'live', index: -1, key: 'live' };
}

function newHistoryTarget(index, lap = lapsData[index]) {
  return { type: 'history', index, key: lapIdentity(lap) };
}

function normalizeSelectedTarget() {
  if (selectedTarget.type === 'live') {
    return;
  }
  if (selectedTarget.type === 'history') {
    const index = findSelectedHistoryIndex();
    if (index >= 0) {
      selectedTarget = newHistoryTarget(index);
      saveSelectedTarget();
      return;
    }
  }
  if (liveLap) {
    selectedTarget = newLiveTarget();
    saveSelectedTarget();
    return;
  }
  if (lapsData.length > 0) {
    const fallbackIndex = Math.min(Math.max(selectedTarget.index, 0), lapsData.length - 1);
    selectedTarget = newHistoryTarget(fallbackIndex);
    saveSelectedTarget();
    return;
  }
  selectedTarget = { type: 'none', index: -1, key: '' };
  saveSelectedTarget();
}

function getSelectedLapIndex(selectableLaps) {
  normalizeSelectedTarget();
  if (selectableLaps.length === 0) {
    return -1;
  }
  if (selectedTarget.type === 'live' && liveLap) {
    return selectableLaps.length - 1;
  }
  const historyIndex = findSelectedHistoryIndex();
  if (historyIndex >= 0) {
    return historyIndex;
  }
  return fallbackHistoryIndex();
}

function loadSelectedTarget() {
  const saved = localStorage.getItem(SELECTED_TARGET_STORAGE_KEY);
  if (!saved) {
    return newLiveTarget();
  }
  try {
    const target = JSON.parse(saved);
    if (target && target.type === 'live') {
      return newLiveTarget();
    }
    if (target && target.type === 'history') {
      return {
        type: 'history',
        index: Number.isInteger(target.index) ? target.index : -1,
        key: typeof target.key === 'string' ? target.key : '',
      };
    }
  } catch (error) {
    console.warn('load selected target failed', error);
  }
  return newLiveTarget();
}

function saveSelectedTarget() {
  if (selectedTarget.type === 'none') {
    localStorage.removeItem(SELECTED_TARGET_STORAGE_KEY);
    return;
  }
  localStorage.setItem(SELECTED_TARGET_STORAGE_KEY, JSON.stringify(selectedTarget));
}

function findSelectedHistoryIndex() {
  if (selectedTarget.type !== 'history') {
    return -1;
  }
  if (selectedTarget.key) {
    const keyIndex = lapsData.findIndex(lap => lapIdentity(lap) === selectedTarget.key);
    if (keyIndex >= 0) {
      return keyIndex;
    }
  }
  if (selectedTarget.index >= 0 && selectedTarget.index < lapsData.length) {
    return selectedTarget.index;
  }
  return -1;
}

function isHistoryLapSelected(index) {
  if (selectedTarget.type === 'history') {
    return findSelectedHistoryIndex() === index;
  }
  return selectedTarget.type === 'live' && !liveLap && fallbackHistoryIndex() === index;
}

function isLiveLapSelected() {
  return selectedTarget.type === 'live' && !!liveLap;
}

function fallbackHistoryIndex() {
  return lapsData.length > 0 ? 0 : -1;
}

function lapIdentity(lap) {
  if (!lap) return '';
  return [
    lap.lap_start_timestamp || '',
    lap.lap_end_timestamp || '',
    lap.number || '',
    lap.lap_finish_time || '',
    lap.car_id || '',
    lap.circuit_id || '',
  ].join('|');
}

function updatePS5Status(isConnected) {
  const connected = !!isConnected;
  if (lastPS5Connected === connected) return;
  lastPS5Connected = connected;

  const dot = document.getElementById('ps5-status-connected');
  const text = document.getElementById('ps5-status-text');
  if (!dot || !text) return;

  dot.className = 'status-dot ' + (connected ? 'green' : 'yellow');
  text.textContent = connected
    ? i18n.t('status.ps5_live')
    : i18n.t('status.ps5_waiting');
}

// Lap table
function renderLapTable() {
  if (lapsData.length === 0 && !liveLap) {
    setLapTableHTML('');
    return;
  }

  const validLaps = lapsData.filter(isRankableLap);
  let bestTime = 0, worstTime = 0;
  if (validLaps.length > 0) {
    bestTime = validLaps.reduce((a, l) => Math.min(a, l.lap_finish_time), Infinity);
    worstTime = validLaps.reduce((a, l) => Math.max(a, l.lap_finish_time), 0);
  }

  const refLap = getBestLap(lapsData);

  let html = '<table><thead><tr>' +
    '<th>' + i18n.t('table.num') + '</th><th>' + i18n.t('table.time') + '</th><th>' + i18n.t('table.diff') + '</th><th>' + i18n.t('table.fuel') + '</th>' +
    '<th>' + i18n.t('table.thr') + '</th><th>' + i18n.t('table.brk') + '</th><th>' + i18n.t('table.cst') + '</th>' +
    '<th>' + i18n.t('table.spin') + '</th><th>' + i18n.t('table.tb') + '</th><th>' + i18n.t('table.track') + '</th><th>' + i18n.t('table.car') + '</th>' +
    '<th>' + i18n.t('table.note') + '</th><th>' + i18n.t('table.action') + '</th>' +
    '</tr></thead><tbody>';

  const sortedRows = lapsData.map((l, i) => ({ lap: l, idx: i }))
    .sort(compareLapRowsByTime);
  sortedRows.forEach(({ lap: l, idx: i }) => {
    const diff = l.lap_finish_time - (bestTime || 0);
    const diffStr = diff === 0 ? '--' : (diff > 0 ? '+' + msToTime(diff) : '-' + msToTime(-diff));
    let cls = '';
    const isRankable = isRankableLap(l);
    if (isRankable) {
      if (l.lap_finish_time === bestTime) cls = 'best';
      else if (l.lap_finish_time === worstTime) cls = 'worst';
    }
    if (isHistoryLapSelected(i)) cls += (cls ? ' ' : '') + 'selected';

    const notes = [];
    if (l.is_pit_lap) notes.push('<span class="badge pit">' + i18n.t('table.pit') + '</span>');
    if (isRankable && l.lap_finish_time === bestTime) notes.push('<span class="badge best">' + i18n.t('table.fastest') + '</span>');
    else if (isRankable && l.lap_finish_time === worstTime) notes.push('<span class="badge worst">' + i18n.t('table.slowest') + '</span>');
    if (refLap && l === refLap) notes.push('<span class="badge ref">' + i18n.t('table.ref') + '</span>');
    if (isHistoryLapSelected(i)) {
      notes.push('<span class="badge target">' + i18n.t('table.target') + '</span>');
    }

    html += '<tr class="' + cls + '" onclick="selectLap(' + i + ')">';
    html += `<td>${l.number || i + 1}</td>`;
    html += `<td>${msToTime(l.lap_finish_time)}</td>`;
    html += `<td>${diffStr}</td>`;
    html += `<td>${l.fuel_consumed != null ? Math.round(l.fuel_consumed) : '-'}</td>`;
    html += `<td>${l.full_throttle_ticks != null ? Math.round(l.full_throttle_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.full_brake_ticks != null ? Math.round(l.full_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.no_throttle_and_no_brake_ticks != null ? Math.round(l.no_throttle_and_no_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.tires_spinning_ticks != null ? Math.round(l.tires_spinning_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${l.throttle_and_brake_ticks != null ? Math.round(l.throttle_and_brake_ticks / (l.lap_ticks || 1) * 100) : '-'}</td>`;
    html += `<td>${escapeHTML(lapTrackName(l))}</td>`;
    html += `<td>${l.car_name || '-'}</td>`;
    html += `<td>${notes.join(' ')}</td>`;
    html += `<td><button class="lap-delete-btn" onclick="deleteLap(${i}, event)">${i18n.t('misc.delete')}</button></td>`;
    html += '</tr>';
  });

  // Live lap row
  if (liveLap) {
    html += `<tr class="live${isLiveLapSelected() ? ' selected' : ''}" onclick="selectLiveLap()">`;
    html += '<td>' + i18n.t('status.live') + '</td>';
    const lapSecs = liveLap._lap_ticks / 60;
    html += `<td>${Math.floor(lapSecs / 60)}:${(lapSecs % 60).toFixed(1).padStart(4, '0')}</td>`;
    html += '<td>--</td>';
    html += `<td>${liveLap.fuel_consumed != null ? Math.round(liveLap.fuel_consumed) : '-'}</td>`;
    const totalTicks = liveLap._lap_ticks || 1;
    html += `<td>${Math.round((liveLap.full_throttle_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.full_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.no_throttle_and_no_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.tires_spinning_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${Math.round((liveLap.throttle_and_brake_ticks || 0) / totalTicks * 100)}</td>`;
    html += `<td>${escapeHTML(lapTrackName(liveLap))}</td>`;
    html += `<td>${currentVehicleModel || '-'}</td>`;
    html += '<td>' + (isLiveLapSelected() ? '<span class="badge target">' + i18n.t('table.target') + '</span>' : '') + '</td><td></td>';
    html += '</tr>';
  }

  html += '</tbody></table>';
  setLapTableHTML(html);
}

function compareLapRowsByTime(a, b) {
  const timestampA = lapSortTimestamp(a.lap);
  const timestampB = lapSortTimestamp(b.lap);
  if (timestampA !== timestampB) {
    return timestampA - timestampB;
  }
  return (a.lap.number || a.idx + 1) - (b.lap.number || b.idx + 1);
}

function lapSortTimestamp(lap) {
  const endTimestamp = Date.parse(lap.lap_end_timestamp || '');
  if (Number.isFinite(endTimestamp)) {
    return endTimestamp;
  }

  const startTimestamp = Date.parse(lap.lap_start_timestamp || '');
  if (Number.isFinite(startTimestamp)) {
    return startTimestamp;
  }

  return Infinity;
}

function setLapTableHTML(html) {
  ['lap-table-driver', 'lap-table', 'lap-table-race', 'lap-table-raceline'].forEach(id => {
    const container = document.getElementById(id);
    if (container) {
      container.innerHTML = html;
    }
  });
}

function renderDriverDashboard() {
  const dashboard = document.getElementById('driver-dashboard');
  if (!dashboard) return;

  const snap = latestTelemetrySnapshot || {};
  const hasTelemetry = !!latestTelemetrySnapshot;
  dashboard.classList.toggle('is-paused', gamePaused);
  dashboard.classList.toggle('is-finished', isRaceComplete);

  const speed = hasTelemetry ? Math.round(snap.speed || 0) : '--';
  const rpm = hasTelemetry ? Math.round(snap.rpm || 0) : '--';
  const gear = formatGear(snap.gear);
  const suggestedGear = isValidDisplayGear(snap.suggested_gear)
    ? i18n.t('driver.suggested_gear') + ' ' + snap.suggested_gear
    : i18n.t('driver.no_suggested_gear');

  setText('driver-speed-value', speed);
  setText('driver-rpm-value', rpm);
  setText('driver-gear-value', gear);
  setText('driver-suggested-gear', hasTelemetry ? suggestedGear : '--');

  const rpmScale = rpmScaleMax(snap.rpm || 0);
  const rpmPercent = rpmScale > 0 ? (snap.rpm || 0) / rpmScale * 100 : 0;
  setHeightPercent('driver-rpm-fill', clampPercent(rpmPercent), 'width');

  const throttle = clampPercent(snap.throttle || 0);
  const brake = clampPercent(snap.brake || 0);
  setHeightPercent('driver-throttle-fill', throttle, 'height');
  setHeightPercent('driver-brake-fill', brake, 'height');
  setText('driver-throttle-value', hasTelemetry ? Math.round(throttle) + '%' : '--%');
  setText('driver-brake-value', hasTelemetry ? Math.round(brake) + '%' : '--%');

  const totalLaps = snap.total_laps > 0 ? snap.total_laps : '?';
  setText('driver-current-lap', hasTelemetry && snap.current_lap > 0 ? `${snap.current_lap}/${totalLaps}` : '--');
  const liveDiffMs = latestTimeDiffMs(liveLap && liveLap.time_diff);
  setText('driver-live-time', liveLap ? formatLapTicks(liveLap._lap_ticks || liveLap.lap_ticks || 0) : '--');
  setText('driver-live-diff', liveDiffMs != null ? formatSignedMs(liveDiffMs) : '--');
  setDriverDiffClass(liveDiffMs);

  setText('driver-last-lap', snap.last_laptime > 0 ? msToTime(snap.last_laptime) : '--');
  setText('driver-best-lap', snap.best_laptime > 0 ? msToTime(snap.best_laptime) : '--');
  setText('driver-fuel', formatFuel(snap.fuel, snap.fuel_capacity));
  setText('driver-boost', hasTelemetry ? Math.max(0, snap.boost || 0).toFixed(2) + ' bar' : '--');
  setText('driver-tire-slip', hasTelemetry ? formatTireSlip(snap.tire_slip) : '--');
  setText('driver-water-temp', hasTelemetry ? Math.round(snap.water_temp || 0) + '°C' : '--');
  setText('driver-oil-temp', hasTelemetry ? Math.round(snap.oil_temp || 0) + '°C' : '--');

  const renderTyre = (id, temp) => {
    setText(id, formatTemperature(temp, hasTelemetry));
    setTemperatureColor(id, temp, hasTelemetry);
  };
  renderTyre('driver-tyre-fl', snap.tyre_temp_fl);
  renderTyre('driver-tyre-fr', snap.tyre_temp_fr);
  renderTyre('driver-tyre-rl', snap.tyre_temp_rl);
  renderTyre('driver-tyre-rr', snap.tyre_temp_rr);
}

function setTemperatureColor(id, temp, hasTelemetry) {
  const el = document.getElementById(id);
  if (!el) return;
  if (hasTelemetry && Number.isFinite(temp) && temp > 0) {
    el.style.color = tireTemperatureColor(temp);
  } else {
    el.style.color = '';
  }
}

function tireTemperatureColor(temp) {
  // Green -> Yellow -> Red with smooth gradient
  // Below 60: green, 60-80: green->yellow, 80-90: yellow->red, 90+: red
  if (temp >= 90) return '#e94560';
  if (temp >= 80) return lerpColor('#f6c85f', '#e94560', (temp - 80) / 10);
  if (temp >= 60) return lerpColor('#19c37d', '#f6c85f', (temp - 60) / 20);
  return '#19c37d';
}

function lerpColor(c1, c2, t) {
  const r1 = parseInt(c1.slice(1,3), 16), g1 = parseInt(c1.slice(3,5), 16), b1 = parseInt(c1.slice(5,7), 16);
  const r2 = parseInt(c2.slice(1,3), 16), g2 = parseInt(c2.slice(3,5), 16), b2 = parseInt(c2.slice(5,7), 16);
  return `rgb(${Math.round(r1+(r2-r1)*t)},${Math.round(g1+(g2-g1)*t)},${Math.round(b1+(b2-b1)*t)})`;
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function setHeightPercent(id, percent, property) {
  const el = document.getElementById(id);
  if (!el) return;
  el.style[property] = `${percent}%`;
}

function setDriverDiffClass(value) {
  const el = document.getElementById('driver-live-diff');
  if (!el) return;
  el.classList.remove('gain', 'loss');
  if (value == null || value === 0) return;
  el.classList.add(value < 0 ? 'gain' : 'loss');
}

function formatGear(value) {
  if (value == null) return '--';
  if (value <= 0 || value === 15) return 'N';
  return String(value);
}

function isValidDisplayGear(value) {
  return Number.isInteger(value) && value > 0 && value < 15;
}

function formatLapTicks(ticks) {
  if (!ticks || ticks <= 0) return '--';
  return msToTime(ticks / 60 * 1000);
}

function formatSignedMs(value) {
  if (!value) return '0.000';
  const sign = value > 0 ? '+' : '-';
  return sign + msToTime(Math.abs(value));
}

function latestTimeDiffMs(timeDiff) {
  if (typeof timeDiff === 'number' && Number.isFinite(timeDiff)) {
    return timeDiff;
  }
  if (!timeDiff || !Array.isArray(timeDiff.timedelta) || timeDiff.timedelta.length === 0) {
    return null;
  }
  const value = timeDiff.timedelta[timeDiff.timedelta.length - 1];
  return Number.isFinite(value) ? value : null;
}

function formatFuel(fuel, capacity) {
  if (fuel == null || !Number.isFinite(fuel)) return '--';
  if (capacity > 0) {
    const percent = Math.round(fuel / capacity * 100);
    return `${Math.round(fuel)}L · ${percent}%`;
  }
  return `${Math.round(fuel)}L`;
}

function formatTireSlip(value) {
  if (value == null || !Number.isFinite(value)) return '--';
  return (value / 4).toFixed(2);
}

function formatTemperature(value, hasTelemetry) {
  if (!hasTelemetry || value == null || !Number.isFinite(value)) return '--';
  return Math.round(value) + '°C';
}

function clampPercent(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

function rpmScaleMax(currentRpm) {
  let maxRPM = currentRpm || 0;
  const selectableLaps = getSelectableLaps();
  for (const lap of selectableLaps) {
    if (!lap || !lap.data_rpm) continue;
    for (const value of lap.data_rpm) {
      if (value > maxRPM) maxRPM = value;
    }
  }
  const rounded = Math.ceil(maxRPM / 1000) * 1000;
  return Math.min(Math.max(rounded, 8000), 14000);
}

function lapTrackName(lap) {
  if (!lap) return '-';

  const name = lap.circuit_name || lap.circuit_id || '-';
  const variation = lap.circuit_variation || '';
  const displayVariation = compactCircuitVariation(name, variation);
  if (!displayVariation || displayVariation === name) {
    return name;
  }
  return `${name} / ${displayVariation}`;
}

function compactCircuitVariation(name, variation) {
  if (!name || !variation || variation === name) {
    return variation;
  }

  const prefix = `${name} - `;
  if (variation.startsWith(prefix)) {
    return variation.slice(prefix.length).trim();
  }

  if (variation.startsWith(name)) {
    return variation.slice(name.length).replace(/^[-:/(\\\s]+/, '').trim();
  }

  return variation;
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || response.statusText);
  }
  return response.json();
}

async function refreshLaps() {
  const data = await fetchJSON('/api/laps');
  lapsData = data.laps || [];
  updateAllCharts();
}

async function refreshLapFiles() {
  const select = document.getElementById('lap-file-select');
  if (!select) return;

  try {
    const current = select.value;
    const data = await fetchJSON('/api/lap-files');
    const files = data.files || [];
    select.innerHTML = files.map(file => {
      const date = file.saved_at ? new Date(file.saved_at).toLocaleString() : '';
      const label = `${date} · ${file.label || file.filename}`;
      return `<option value="${escapeHTML(file.filename)}">${escapeHTML(label)}</option>`;
    }).join('');
    if (current && files.some(file => file.filename === current)) {
      select.value = current;
    }
  } catch (error) {
    console.error('refresh lap files failed', error);
  }
}

async function saveAllLaps() {
  try {
    await fetchJSON('/api/lap-files/save', { method: 'POST', body: '{}' });
    liveLap = null;
    currentLiveLapNum = -1;
    await refreshLapFiles();
    await refreshLaps();
  } catch (error) {
    alert(error.message);
  }
}

async function loadSelectedLapFile() {
  const select = document.getElementById('lap-file-select');
  if (!select || !select.value) return;

  try {
    await fetchJSON('/api/lap-files/load', {
      method: 'POST',
      body: JSON.stringify({ filename: select.value }),
    });
    await refreshLaps();
  } catch (error) {
    alert(error.message);
  }
}

async function deleteSelectedLapFile() {
  const select = document.getElementById('lap-file-select');
  if (!select || !select.value) return;
  if (!confirm(i18n.t('misc.confirm_delete_file'))) {
    return;
  }

  try {
    await fetchJSON('/api/lap-files', {
      method: 'DELETE',
      body: JSON.stringify({ filename: select.value }),
    });
    await refreshLapFiles();
  } catch (error) {
    alert(error.message);
  }
}

async function clearAllLaps() {
  if (!confirm(i18n.t('misc.confirm_clear'))) {
    return;
  }

  try {
    await fetchJSON('/api/laps/clear', { method: 'POST', body: '{}' });
    liveLap = null;
    currentLiveLapNum = -1;
    await refreshLaps();
  } catch (error) {
    alert(error.message);
  }
}

async function deleteLap(index, event) {
  if (event) event.stopPropagation();

  try {
    await fetchJSON('/api/laps', {
      method: 'DELETE',
      body: JSON.stringify({ indices: [index] }),
    });
    await refreshLaps();
  } catch (error) {
    alert(error.message);
  }
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, char => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  }[char]));
}

function renderShiftAnalysis(laps, idx) {
  const container = document.getElementById('rpm-peaks-table');
  if (!container) return;
  const lap = laps[idx] || laps[0];
  if (!lap || !lap.data_gear || !lap.data_rpm || lap.data_gear.length < 10) {
    container.innerHTML = '<div style="color:#666;padding:10px;font-size:12px">' + i18n.t('shift.no_data') + '</div>';
    return;
  }

  // Detect upshift events: gear increases, skipping neutral (0)
  const shifts = [];
  let prevGear = 0;
  let preRPM = 0;
  for (let i = 0; i < lap.data_gear.length; i++) {
    const g = lap.data_gear[i];
    const r = lap.data_rpm[i];
    if (g <= 0) { prevGear = 0; continue; }
    if (prevGear > 0 && g > prevGear) {
      shifts.push({ fromGear: prevGear, toGear: g, preRPM: Math.round(preRPM), postRPM: Math.round(r) });
    }
    prevGear = g;
    preRPM = r;
  }

  if (shifts.length === 0) {
    container.innerHTML = '<div style="color:#666;padding:10px;font-size:12px">' + i18n.t('shift.no_data') + '</div>';
    return;
  }

  const avgRPM = Math.round(shifts.reduce((s, x) => s + x.preRPM, 0) / shifts.length);
  let html = '<div class="section-title">' + i18n.t('shift.title') + '</div>';
  html += '<table><thead><tr>' +
    '<th>' + i18n.t('shift.gear') + '</th>' +
    '<th>' + i18n.t('shift.pre_rpm') + '</th>' +
    '<th>' + i18n.t('shift.post_rpm') + '</th>' +
    '<th>' + i18n.t('shift.drop') + '</th>' +
    '<th>' + i18n.t('shift.drop_pct') + '</th>' +
    '</tr></thead><tbody>';
  for (const s of shifts) {
    const drop = s.preRPM - s.postRPM;
    const pct = s.preRPM > 0 ? Math.round(drop / s.preRPM * 100) : 0;
    html += '<tr class="' + (drop < 0 ? 'shift-anomaly' : '') + '">' +
      `<td>${s.fromGear}→${s.toGear}</td>` +
      `<td>${s.preRPM}</td>` +
      `<td>${s.postRPM}</td>` +
      `<td>${drop}</td>` +
      `<td>${pct}%</td>` +
      '</tr>';
  }
  html += '</tbody></table>';
  // Summary
  html += '<div class="shift-summary">' + shifts.length + ' ' + i18n.t('table.num').toLowerCase() + ' · ' +
    i18n.t('shift.avg_rpm') + ': ' + avgRPM + '</div>';
  container.innerHTML = html;
}

function getBestLap(laps) {
  let best = null;
  let bestTime = Infinity;
  for (const l of laps) {
    if (!isRankableLap(l)) continue;
    if (l.lap_finish_time < bestTime) { bestTime = l.lap_finish_time; best = l; }
  }
  return best;
}

function isRankableLap(lap) {
  if (!lap || lap._is_live || lap.is_pit_lap || !lap.lap_finish_time || lap.lap_finish_time <= 0) {
    return false;
  }
  return lap.is_complete === true;
}

function msToTime(ms) {
  const secs = ms / 1000;
  const mins = Math.floor(secs / 60);
  const remain = secs - mins * 60;
  return `${mins}:${remain.toFixed(3).padStart(6, '0')}`;
}

// Replay record toggle
ws.on('replay_record_state', (data) => {
  const enabled = data.enabled || false;
  document.getElementById('replay-record-toggle').checked = enabled;
  document.getElementById('replay-record-container').style.display = '';
});

function toggleReplayRecord(enabled) {
  ws.send('set_replay_record', null, { enabled });
}

// Init on load
document.addEventListener('DOMContentLoaded', () => {
  i18n.apply();
  const targetLang = i18n.lang === 'zh' ? 'en' : 'zh';
  document.getElementById('lang-toggle').textContent = i18n.t('lang.' + targetLang);
  refreshLapFiles();
  initCharts();
});
