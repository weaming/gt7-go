registerChart('speed', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['speed'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.speed'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value' },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1 }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx, liveLap) {
    const chart = charts['speed'];
    if (!chart) return;
    const targetLap = liveLap || laps[idx] || laps[0];
    if (!targetLap || !targetLap.data_speed) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const x = xAxis(targetLap.data_speed);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== targetLap && best.data_speed) {
      const bx = xAxis(best.data_speed);
      bestData = best.data_speed.map((v, i) => [bx[i], v]);
    }
    chart.setOption({
      xAxis: { show: true },
      series: [
        { data: targetLap.data_speed.map((v, i) => [x[i], v]) },
        { data: bestData },
      ],
    });
  }
});

// dt = tick interval in seconds. data arrays can carry _tickInterval
// for non-60fps recordings (e.g. old 10fps data loaded from file).
function xAxis(speed) {
  const dt = speed._tickInterval || (1 / 60);
  const result = [0];
  for (let i = 1; i < speed.length; i++) {
    result.push(result[i-1] + (speed[i] / 3.6 / 1000) * dt);
  }
  return result;
}
