// Track map with corner markers
registerChart('corner-track', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['corner-track'] = chart;
    chart.setOption({
      title: { text: i18n.t('tab.corner'), textStyle: { fontSize: 12 } },
      tooltip: {
        trigger: 'item',
        formatter(params) {
          if (params.seriesIndex !== 1 || !params.data) return ' ';
          const idx = params.dataIndex;
          const result = window._cornerResult;
          if (!result) return ' ';
          const c = result.corners[idx];
          if (!c) return ' ';
          return '<b>' + c.label + '</b><br/>'
            + i18n.t('misc.time_lost') + ': +' + roundMs(c.timeLost) + '<br/>'
            + 'Speed: ' + Math.round(c.entrySpeed) + ' → ' + Math.round(c.minSpeed) + ' km/h';
        },
      },
      grid: { left: 10, right: 10, top: 35, bottom: 10 },
      xAxis: { type: 'value', show: false, scale: true },
      yAxis: { type: 'value', show: false, scale: true },
      series: [
        {
          name: 'Track',
          type: 'line',
          data: [],
          smooth: true,
          showSymbol: false,
          lineStyle: { color: '#555', width: 2 },
        },
        {
          name: 'Corners',
          type: 'scatter',
          data: [],
          label: { show: true, formatter: '{b}', fontSize: 10, fontWeight: 'bold', color: '#fff' },
          emphasis: {
            scale: 1.8,
            label: { fontWeight: 'bold', fontSize: 12 },
            itemStyle: { borderColor: '#fff', borderWidth: 3 },
          },
        },
      ],
    });
  },
  update(laps, idx, liveLap) {
    const chart = charts['corner-track'];
    if (!chart) return;
    if (laps.length < 2) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }

    const lap = liveLap || laps[idx];
    const result = cornerTimeLoss(laps, lap ? laps.indexOf(lap) : idx);
    if (!result || !result.corners || result.corners.length === 0) {
      chart.setOption({ series: [{ data: [] }, { data: [] }] });
      return;
    }

    // Use best lap position data for track outline
    const best = getBestLap(laps);
    if (!best || !best.data_position_x || !best.data_position_z) return;

    const posX = best.data_position_x;
    const posZ = best.data_position_z;
    const track = posX.map((x, i) => [x, -posZ[i]]);

    const dist = xAxis(best.data_speed);

    // Corner markers at apex positions
    const markers = result.corners.map((c, i) => {
      let px = 0, pz = 0;
      for (let j = 0; j < dist.length; j++) {
        if (dist[j] >= c.apexDist) { px = posX[j]; pz = posZ[j]; break; }
      }

      const lost = c.timeLost;
      let color = '#66bb6a';
      if (lost > 30) color = '#e94560';
      else if (lost > 15) color = '#ff9800';
      else if (lost > 5) color = '#ffd54f';

      return {
        value: [px, -pz],
        name: (i + 1).toString(),
        itemStyle: { color, borderColor: '#fff', borderWidth: 1 },
        symbolSize: Math.max(16, Math.min(32, 16 + lost / 3)),
      };
    });

    chart.setOption({
      series: [
        { data: track },
        { data: markers },
      ],
    });

    // Hover: track marker -> highlight bar chart
    chart.off('mouseover');
    chart.on('mouseover', function(params) {
      if (params.componentType === 'series' && params.seriesIndex === 1) {
        const barChart = charts['corner'];
        if (barChart) {
          barChart.dispatchAction({ type: 'downplay' });
          barChart.dispatchAction({ type: 'highlight', seriesIndex: 0, dataIndex: params.dataIndex });
        }
      }
    });
    chart.off('mouseout');
    chart.on('mouseout', function() {
      const barChart = charts['corner'];
      if (barChart) barChart.dispatchAction({ type: 'downplay' });
    });
  }
});

// Time loss per corner bar chart
registerChart('corner', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['corner'] = chart;
    chart.setOption({
      title: { text: i18n.t('misc.time_lost'), textStyle: { fontSize: 12 } },
      tooltip: {
        trigger: 'axis',
        formatter(params) {
          const p = params[0];
          if (!p || p.value == null) return '';
          const idx = p.dataIndex;
          const result = window._cornerResult;
          if (!result) return '';
          const c = result.corners[idx];
          if (!c) return '';
          return '<b>' + c.label + '</b><br/>'
            + i18n.t('misc.time_lost') + ': +' + roundMs(c.timeLost) + '<br/>'
            + 'Speed: ' + Math.round(c.entrySpeed) + ' → ' + Math.round(c.minSpeed) + ' km/h<br/>'
            + 'Distance: ' + Math.round(c.startDist * 1000) + '–' + Math.round(c.endDist * 1000) + 'm';
        },
      },
      grid: { left: 55, right: 100, top: 35, bottom: 55 },
      xAxis: { type: 'category', data: [], axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value' },
      series: [
        {
          type: 'bar',
          barCategoryGap: '40%',
          data: [],
          label: { show: true, position: 'top', fontSize: 10, fontWeight: 'bold' },
          itemStyle: { borderRadius: [3, 3, 0, 0] },
          emphasis: {
            itemStyle: {
              shadowBlur: 20,
              shadowColor: 'rgba(255,255,255,0.6)',
              borderColor: '#fff',
              borderWidth: 2,
            },
            label: { fontWeight: 'bold', fontSize: 12 },
          },
        },
      ],
    });
  },
  update(laps, idx, liveLap) {
    const chart = charts['corner'];
    if (!chart) return;
    if (laps.length < 2) { chart.setOption({ series: [{ data: [] }], graphic: [] }); return; }

    const lap = liveLap || laps[idx];
    const result = cornerTimeLoss(laps, lap ? laps.indexOf(lap) : idx);
    window._cornerResult = result;
    if (!result || !result.corners || result.corners.length === 0) {
      chart.setOption({ series: [{ data: [] }], graphic: [] });
      return;
    }

    const totalGain = result.corners.reduce((s, c) => s + c.timeLost, 0);
    const bestLap = getBestLap(laps);
    if (!bestLap) {
      chart.setOption({ series: [{ data: [] }], graphic: [] });
      return;
    }

    // Summary text
    const summaryText = i18n.t('misc.best_lap') + ': ' + msToTime(bestLap.lap_finish_time)
      + '  |  ' + i18n.t('misc.potential') + ': {gain|-' + msToTime(totalGain) + '}';

    // Sort corners by time lost for top-N display
    const sorted = [...result.corners].sort((a, b) => b.timeLost - a.timeLost);
    const topLoss = sorted.slice(0, 3);
    let topText = '';
    if (topLoss.length > 0) {
      topText = '{title|' + i18n.t('misc.potential') + '}\n'
        + topLoss.map((c, i) => {
          const pct = totalGain > 0 ? Math.round(c.timeLost / totalGain * 100) : 0;
          return '#' + (i + 1) + '  +' + roundMs(c.timeLost) + ' (' + pct + '%)';
        }).join('\n');
    }

    const chartData = result.corners.map((c) => {
      const lost = c.timeLost;
      let color = '#66bb6a';
      if (lost > 30) color = '#e94560';
      else if (lost > 15) color = '#ff9800';
      else if (lost > 5) color = '#ffd54f';

      return {
        value: lost,
        itemStyle: { color },
        label: {
          show: lost > 0,
          formatter: lost > 0 ? '+' + roundMs(lost) : '',
          color: lost > 15 ? '#e94560' : '#aaa',
          fontSize: 10,
        },
      };
    });

    chart.setOption({
      xAxis: {
        data: result.corners.map(c =>
          c.label + '\n' + Math.round(c.entrySpeed) + '→' + Math.round(c.minSpeed)
        ),
      },
      series: [{ data: chartData }],
      graphic: [
        {
          type: 'text', left: 55, top: 4,
          style: {
            fill: '#aaa', fontSize: 10,
            text: summaryText,
            rich: { gain: { fill: '#e94560', fontSize: 10 } },
          },
        },
        {
          type: 'text', right: 20, top: 32,
          style: {
            fill: '#aaa', fontSize: 11, lineHeight: 18,
            text: topText,
            rich: {
              title: { fill: '#e94560', fontWeight: 'bold', fontSize: 11 },
            },
          },
        },
      ],
    });

    // Hover: bar/axis -> highlight track marker & bar chart
    chart.off('mouseover');
    chart.on('mouseover', function(params) {
      if (params.dataIndex != null) {
        chart.dispatchAction({ type: 'downplay' });
        chart.dispatchAction({ type: 'highlight', seriesIndex: 0, dataIndex: params.dataIndex });
        const trackChart = charts['corner-track'];
        if (trackChart) {
          trackChart.dispatchAction({ type: 'downplay' });
          trackChart.dispatchAction({ type: 'highlight', seriesIndex: 1, dataIndex: params.dataIndex });
        }
      }
    });
    chart.off('mouseout');
    chart.on('mouseout', function() {
      chart.dispatchAction({ type: 'downplay' });
      const trackChart = charts['corner-track'];
      if (trackChart) trackChart.dispatchAction({ type: 'downplay' });
    });
  }
});

// ---- Shared analysis functions ----

function roundMs(ms) {
  if (ms < 1) return '<1ms';
  return Math.round(ms) + 'ms';
}

function cornerTimeLoss(laps, idx) {
  const referenceLap = getBestLap(laps);
  const targetLap = laps[idx];
  if (referenceLap && targetLap && targetLap !== referenceLap && targetLap.data_speed && targetLap.data_speed.length > 0) {
    return targetCornerTimeLoss(referenceLap, targetLap);
  }

  const filtered = laps.filter(l => isRankableLap(l) && l.data_speed && l.data_speed.length > 0)
    .sort((a, b) => a.lap_finish_time - b.lap_finish_time);
  if (filtered.length < 2) return null;

  const best = referenceLap || filtered[0];
  const speed = best.data_speed;
  const dist = xAxis(speed);
  if (!dist || dist.length < 20) return null;

  // Smooth speed to reduce noise
  const window = 5;
  const smoothed = [];
  for (let i = 0; i < speed.length; i++) {
    const start = Math.max(0, i - window);
    const end = Math.min(speed.length, i + window + 1);
    let sum = 0;
    for (let j = start; j < end; j++) sum += speed[j];
    smoothed.push(sum / (end - start));
  }

  // Find local minima (valleys = corner apexes)
  const valleys = [];
  for (let i = 3; i < smoothed.length - 3; i++) {
    if (smoothed[i] < smoothed[i - 1] && smoothed[i] < smoothed[i + 1] &&
        smoothed[i] < smoothed[i - 2] && smoothed[i] < smoothed[i + 2]) {
      valleys.push(i);
    }
  }

  // For each valley, find preceding peak (braking start)
  const rawCorners = [];
  for (const vi of valleys) {
    let peakSpeed = 0;
    let peakIdx = vi;
    for (let j = Math.max(0, vi - 90); j < vi; j++) {
      if (smoothed[j] > peakSpeed) { peakSpeed = smoothed[j]; peakIdx = j; }
    }
    const drop = peakSpeed - smoothed[vi];
    const brakeDist = dist[vi] - dist[peakIdx];
    if (drop > 10 && brakeDist > 0.015) {
      rawCorners.push({
        label: '',
        startDist: dist[peakIdx],
        apexIdx: vi,
        apexDist: dist[vi],
        endDist: dist[vi] + brakeDist * 0.8,
        entrySpeed: peakSpeed,
        minSpeed: smoothed[vi],
        speedDrop: drop,
      });
    }
  }

  if (rawCorners.length === 0) return null;

  // Merge close corners (e.g., chicanes)
  const merged = [rawCorners[0]];
  for (let i = 1; i < rawCorners.length; i++) {
    const last = merged[merged.length - 1];
    if (rawCorners[i].startDist - last.endDist < 0.03) {
      last.endDist = rawCorners[i].endDist;
      last.minSpeed = Math.min(last.minSpeed, rawCorners[i].minSpeed);
      last.speedDrop = Math.max(last.speedDrop, rawCorners[i].speedDrop);
      last.apexIdx = smoothed[rawCorners[i].apexIdx] < smoothed[last.apexIdx] ? rawCorners[i].apexIdx : last.apexIdx;
      last.apexDist = dist[last.apexIdx];
    } else {
      merged.push(rawCorners[i]);
    }
  }

  const corners = merged.slice(0, 8);

  // Compute time loss per corner across fast laps
  const fastLaps = filtered.slice(0, 5);
  const threshold = fastLaps[0].lap_finish_time * 1.05;
  const analysisLaps = fastLaps.filter(l => l.lap_finish_time <= threshold);
  if (analysisLaps.length < 2) return null;

  const lapDists = analysisLaps.map(l => xAxis(l.data_speed));
  const lapTimes = analysisLaps.map(l => l.data_speed.map((_, j) => j / 60));

  corners.forEach((c, ci) => {
    let minTime = Infinity;
    let bestTime = 0;
    const start = Math.max(0, c.startDist - 0.005);
    const end = c.endDist;

    for (let li = 0; li < analysisLaps.length; li++) {
      const d = lapDists[li];
      const t = lapTimes[li];
      if (!d || d.length === 0) continue;
      const e = Math.min(end, d[d.length - 1]);
      const st = interp(d, t, start);
      const et = interp(d, t, e);
      const segTime = Math.max(0, et - st);
      if (segTime > 0 && segTime < minTime) minTime = segTime;
      if (li === 0) bestTime = segTime;
    }

    c.theoreticalTime = minTime < Infinity ? minTime * 1000 : 0;
    c.actualTime = bestTime * 1000;
    c.timeLost = Math.max(0, c.actualTime - c.theoreticalTime);
    c.label = i18n.t('misc.segment') + ' ' + (ci + 1);
  });

  return { corners };
}

function targetCornerTimeLoss(referenceLap, targetLap) {
  const corners = detectCorners(referenceLap);
  if (!corners || corners.length === 0) return null;

  const refDist = xAxis(referenceLap.data_speed);
  const targetDist = xAxis(targetLap.data_speed);
  const targetTimes = targetLap.data_speed.map((_, j) => j / 60);
  const refTimes = referenceLap.data_speed.map((_, j) => j / 60);
  if (!targetDist || targetDist.length < 2 || !refDist || refDist.length < 2) return null;

  const targetMaxDist = targetDist[targetDist.length - 1];
  const completedCorners = [];
  corners.forEach((corner, index) => {
    if (corner.endDist > targetMaxDist) return;

    const start = Math.max(0, corner.startDist - 0.005);
    const end = corner.endDist;
    const refStartTime = interp(refDist, refTimes, start);
    const refEndTime = interp(refDist, refTimes, end);
    const targetStartTime = interp(targetDist, targetTimes, start);
    const targetEndTime = interp(targetDist, targetTimes, end);
    if ([refStartTime, refEndTime, targetStartTime, targetEndTime].some(v => v == null)) return;

    const result = Object.assign({}, corner);
    result.theoreticalTime = Math.max(0, refEndTime - refStartTime) * 1000;
    result.actualTime = Math.max(0, targetEndTime - targetStartTime) * 1000;
    result.timeLost = Math.max(0, result.actualTime - result.theoreticalTime);
    result.label = i18n.t('misc.segment') + ' ' + (index + 1);
    completedCorners.push(result);
  });

  return completedCorners.length > 0 ? { corners: completedCorners } : null;
}

function detectCorners(lap) {
  const speed = lap.data_speed;
  const dist = xAxis(speed);
  if (!dist || dist.length < 20) return null;

  const window = 5;
  const smoothed = [];
  for (let i = 0; i < speed.length; i++) {
    const start = Math.max(0, i - window);
    const end = Math.min(speed.length, i + window + 1);
    let sum = 0;
    for (let j = start; j < end; j++) sum += speed[j];
    smoothed.push(sum / (end - start));
  }

  const valleys = [];
  for (let i = 3; i < smoothed.length - 3; i++) {
    if (smoothed[i] < smoothed[i - 1] && smoothed[i] < smoothed[i + 1] &&
        smoothed[i] < smoothed[i - 2] && smoothed[i] < smoothed[i + 2]) {
      valleys.push(i);
    }
  }

  const rawCorners = [];
  for (const valleyIndex of valleys) {
    let peakSpeed = 0;
    let peakIndex = valleyIndex;
    for (let j = Math.max(0, valleyIndex - 90); j < valleyIndex; j++) {
      if (smoothed[j] > peakSpeed) {
        peakSpeed = smoothed[j];
        peakIndex = j;
      }
    }

    const speedDrop = peakSpeed - smoothed[valleyIndex];
    const brakeDist = dist[valleyIndex] - dist[peakIndex];
    if (speedDrop > 10 && brakeDist > 0.015) {
      rawCorners.push({
        label: '',
        startDist: dist[peakIndex],
        apexIdx: valleyIndex,
        apexDist: dist[valleyIndex],
        endDist: dist[valleyIndex] + brakeDist * 0.8,
        entrySpeed: peakSpeed,
        minSpeed: smoothed[valleyIndex],
        speedDrop,
      });
    }
  }

  if (rawCorners.length === 0) return null;

  const merged = [rawCorners[0]];
  for (let i = 1; i < rawCorners.length; i++) {
    const last = merged[merged.length - 1];
    if (rawCorners[i].startDist - last.endDist < 0.03) {
      last.endDist = rawCorners[i].endDist;
      last.minSpeed = Math.min(last.minSpeed, rawCorners[i].minSpeed);
      last.speedDrop = Math.max(last.speedDrop, rawCorners[i].speedDrop);
      last.apexIdx = smoothed[rawCorners[i].apexIdx] < smoothed[last.apexIdx] ? rawCorners[i].apexIdx : last.apexIdx;
      last.apexDist = dist[last.apexIdx];
    } else {
      merged.push(rawCorners[i]);
    }
  }

  return merged.slice(0, 8);
}
