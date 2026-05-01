registerChart('speed', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['speed'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.speed'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', name: 'Distance (km)', show: false },
      yAxis: { type: 'value', name: 'km/h' },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1 }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['speed'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_speed) return;
    const x = xAxis(lap.data_speed);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_speed) {
      const bx = xAxis(best.data_speed);
      bestData = best.data_speed.map((v, i) => [bx[i], v]);
    }
    chart.setOption({
      xAxis: { show: true },
      series: [
        { data: lap.data_speed.map((v, i) => [x[i], v]) },
        { data: bestData },
      ],
    });
  }
});

function xAxis(speed) {
  const result = [0];
  for (let i = 1; i < speed.length; i++) {
    result.push(result[i-1] + (speed[i] / 3.6 / 1000) * 16.668);
  }
  return result;
}
