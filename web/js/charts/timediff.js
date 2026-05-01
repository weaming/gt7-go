registerChart('timediff', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['timediff'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.timediff'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', name: 'Distance (km)', show: false },
      yAxis: { type: 'value' },
      series: [{ type: 'line', showSymbol: false, lineStyle: { width: 1 }, areaStyle: { opacity: 0.2 }, data: [] }],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['timediff'];
    const best = getBestLap(laps);
    if (!chart || laps.length < 2 || !best) { chart?.setOption({ series: [{ data: [] }] }); return; }
    const ref = best;
    const lap = laps[idx] || laps[0];
    if (!ref.data_speed || !lap.data_speed) return;
    const refDist = xAxis(ref.data_speed), compDist = xAxis(lap.data_speed);
    const maxDist = Math.max(refDist[refDist.length-1], compDist[compDist.length-1]);
    const numPoints = 500;
    const dists = [], deltas = [];
    for (let i = 0; i < numPoints; i++) {
      const d = maxDist * i / (numPoints-1);
      dists.push(d);
      const rt = interp(refDist, ref.data_speed.map((_,j)=>j/60), d);
      const ct = interp(compDist, lap.data_speed.map((_,j)=>j/60), d);
      deltas.push(rt >= 0 && ct >= 0 ? (ct - rt) * 1000 : null);
    }
    chart.setOption({
      xAxis: { show: true },
      series: [{ data: deltas.map((v,i) => [dists[i], v]) }],
    });
  }
});

function interp(dist, times, target) {
  if (target <= dist[0]) return times[0];
  if (target >= dist[dist.length-1]) return times[times.length-1];
  let idx = dist.findIndex(d => d >= target);
  if (idx <= 0) return times[0];
  if (idx >= dist.length) return times[times.length-1];
  const frac = (target - dist[idx-1]) / (dist[idx] - dist[idx-1]);
  return times[idx-1] + frac * (times[idx] - times[idx-1]);
}
