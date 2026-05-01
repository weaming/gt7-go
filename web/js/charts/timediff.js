registerChart('timediff', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['timediff'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.timediff'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value' },
      series: [{ type: 'line', showSymbol: false, lineStyle: { width: 1 }, areaStyle: { opacity: 0.2 }, data: [] }],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['timediff'];
    const best = getBestLap(laps);
    if (!chart || laps.length < 2 || !best) { chart?.setOption({ series: [{ data: [] }] }); return; }

    const lap = laps[idx] || laps[0];
    if (!lap || !best) return;

    // Use backend-computed time diff if available (position-based, more accurate)
    if (lap.time_diff && lap.time_diff.distance && lap.time_diff.timedelta) {
      const data = normalizeTimeDiffDistance(lap.time_diff.distance, best, lap)
        .map((d, i) => [d, lap.time_diff.timedelta[i]]);
      chart.setOption({
        xAxis: { show: true },
        series: [{ data }],
      });
      return;
    }

    // Fallback: client-side computation using speed-based distance estimation
    if (!best.data_speed || !lap.data_speed) return;
    const refDist = xAxis(best.data_speed), compDist = xAxis(lap.data_speed);
    const refDistEnd = refDist[refDist.length - 1];
    const compDistEnd = compDist[compDist.length - 1];
    const maxValidDist = Math.min(refDistEnd, compDistEnd);
    if (maxValidDist <= 0) return;
    const numPoints = 500;
    const dists = [], deltas = [];
    for (let i = 0; i < numPoints; i++) {
      const d = maxValidDist * i / (numPoints-1);
      dists.push(d);
      const rt = interp(refDist, best.data_speed.map((_,j)=>j/60), d);
      const ct = interp(compDist, lap.data_speed.map((_,j)=>j/60), d);
      deltas.push((ct - rt) * 1000);
    }
    chart.setOption({
      xAxis: { show: true },
      series: [{ data: deltas.map((v,i) => [dists[i], v]) }],
    });
  }
});

function normalizeTimeDiffDistance(distance, best, lap) {
  if (!distance || distance.length === 0) return [];

  const lastDistance = distance[distance.length - 1];
  const expectedMax = expectedTimeDiffMaxDistance(best, lap);
  const shouldConvertMeters = expectedMax > 0 ? lastDistance > expectedMax * 5 : lastDistance > 100;
  if (!shouldConvertMeters) return distance;

  return distance.map(d => d / 1000);
}

function expectedTimeDiffMaxDistance(best, lap) {
  const distances = [];
  if (best && best.data_speed) {
    const bestDistance = xAxis(best.data_speed);
    if (bestDistance.length > 0) distances.push(bestDistance[bestDistance.length - 1]);
  }
  if (lap && lap.data_speed) {
    const lapDistance = xAxis(lap.data_speed);
    if (lapDistance.length > 0) distances.push(lapDistance[lapDistance.length - 1]);
  }
  if (distances.length === 0) return 0;
  return Math.min(...distances);
}

function interp(dist, times, target) {
  if (target <= dist[0]) return times[0];
  if (target > dist[dist.length-1]) return null;
  let idx = dist.findIndex(d => d >= target);
  if (idx <= 0) return times[0];
  if (idx >= dist.length) return times[times.length-1];
  const frac = (target - dist[idx-1]) / (dist[idx] - dist[idx-1]);
  return times[idx-1] + frac * (times[idx] - times[idx-1]);
}
