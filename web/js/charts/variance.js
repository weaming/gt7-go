registerChart('variance', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['variance'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.variance'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value' },
      series: [{ type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff5722' }, areaStyle: { opacity: 0.2 }, data: [] }],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx, liveLap) {
    const chart = charts['variance'];
    if (!chart || laps.length < 2) { chart?.setOption({ series: [{ data: [] }] }); return; }
    const variance = computeVariance(laps);
    if (!variance || !variance.distance) return;
    chart.setOption({
      xAxis: { show: true },
      series: [{ data: variance.distance.map((d, i) => [d, variance.speed_variance[i] || 0]) }],
    });
  }
});

function computeVariance(laps) {
  if (laps.length < 2) return null;
  const numBins = 500;
  let maxDist = 0;
  const dists = laps.map(l => {
    const d = xAxis(l.data_speed);
    if (d.length > 0 && d[d.length-1] > maxDist) maxDist = d[d.length-1];
    return d;
  });
  if (maxDist <= 0) return null;
  const binSize = maxDist / numBins;
  const distances = [], variances = [];
  for (let bin = 0; bin < numBins; bin++) {
    const binStart = bin * binSize, binEnd = binStart + binSize;
    distances.push((binStart + binEnd) / 2);
    const speeds = [];
    laps.forEach((l, li) => {
      for (let j = 0; j < l.data_speed.length; j++) {
        if (dists[li][j] >= binStart && dists[li][j] < binEnd) speeds.push(l.data_speed[j]);
      }
    });
    if (speeds.length >= 2) {
      const mean = speeds.reduce((a, b) => a + b, 0) / speeds.length;
      const sqDiff = speeds.reduce((sum, v) => sum + (v - mean) ** 2, 0);
      variances.push(Math.sqrt(sqDiff / (speeds.length - 1)));
    } else {
      variances.push(0);
    }
  }
  return { distance: distances, speed_variance: variances };
}
