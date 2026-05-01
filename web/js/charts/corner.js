registerChart('corner', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['corner'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.corner'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      legend: { textStyle: { color: '#aaa' }, top: 25 },
      xAxis: { type: 'value', name: 'Distance (km)' },
      yAxis: [
        { type: 'value', name: i18n.t('misc.time_ms') },
        { type: 'value', name: i18n.t('misc.stddev') },
      ],
      series: [
        { name: i18n.t('misc.theoretical_best'), type: 'bar', data: [] },
        { name: i18n.t('misc.best_lap'), type: 'bar', data: [] },
        { name: i18n.t('misc.consistency'), type: 'line', yAxisIndex: 1, data: [] },
      ],
      grid: { left: 50, right: 50, top: 55, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['corner'];
    if (!chart) return;
    if (laps.length < 2) { chart.setOption({ series: [{ data: [] }, { data: [] }, { data: [] }] }); return; }
    const result = segmentAnalysis(laps);
    if (!result) return;
    const bestLap = laps.slice().sort((a, b) => a.lap_finish_time - b.lap_finish_time)[0];
    chart.setOption({
      xAxis: { data: result.segment_distances.map(d => Math.round(d * 1000) / 1000) },
      series: [
        { data: result.theoretical_times },
        { data: result.best_lap_times },
        { data: result.consistency_stddev },
      ],
    });
  }
});

function segmentAnalysis(laps) {
  const filtered = laps.filter(l => !l.is_replay && l.data_speed && l.data_speed.length > 0)
    .sort((a, b) => a.lap_finish_time - b.lap_finish_time)
    .slice(0, 5);
  if (filtered.length < 2) return null;
  const threshold = filtered[0].lap_finish_time * 1.05;
  const fastLaps = filtered.filter(l => l.lap_finish_time <= threshold).slice(0, 5);
  if (fastLaps.length < 2) return null;
  const numSegments = 20;
  const dists = fastLaps.map(l => xAxis(l.data_speed));
  let maxDist = 0;
  dists.forEach(d => { if (d.length > 0 && d[d.length-1] > maxDist) maxDist = d[d.length-1]; });
  if (maxDist <= 0) return null;
  const segSize = maxDist / numSegments;
  const result = { num_segments: numSegments, segment_distances: [], consistency_stddev: [], theoretical_times: [], best_lap_times: [], theoretical_total: 0, best_lap_total: 0, potential_gain: 0 };
  const best = fastLaps[0];
  const bestIdx = 0;
  for (let seg = 0; seg < numSegments; seg++) {
    const segStart = seg * segSize, segEnd = segStart + segSize;
    result.segment_distances.push((segStart + segEnd) / 2);
    const speeds = [];
    let minSegTime = Infinity, bestSegTime = 0;
    fastLaps.forEach((l, li) => {
      const d = dists[li], times = l.data_speed.map((_, j) => j / 60);
      const st = interp(d, times, segStart), et = interp(d, times, segEnd);
      const segTime = et - st;
      if (segTime > 0 && segTime < minSegTime) minSegTime = segTime;
      if (li === 0) bestSegTime = segTime;
      l.data_speed.forEach((s, j) => { if (j < d.length && d[j] >= segStart && d[j] < segEnd) speeds.push(s); });
    });
    result.theoretical_times.push(minSegTime < Infinity ? minSegTime * 1000 : 0);
    result.best_lap_times.push(bestSegTime * 1000);
    result.theoretical_total += result.theoretical_times[seg];
    result.best_lap_total += result.best_lap_times[seg];
    if (speeds.length >= 2) {
      const mean = speeds.reduce((a, b) => a + b, 0) / speeds.length;
      const sqDiff = speeds.reduce((s, v) => s + (v - mean) ** 2, 0);
      result.consistency_stddev.push(Math.sqrt(sqDiff / (speeds.length - 1)));
    } else {
      result.consistency_stddev.push(0);
    }
  }
  result.potential_gain = Math.max(0, result.best_lap_total - result.theoretical_total);
  return result;
}
