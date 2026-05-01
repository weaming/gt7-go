registerChart('throttle', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['throttle'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.throttle'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value', min: 0, max: 100 },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#4caf50' }, areaStyle: { color: 'rgba(76,175,80,0.2)' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['throttle'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_throttle) return;
    const x = xAxis(lap.data_speed || lap.data_throttle);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_throttle) {
      const bx = xAxis(best.data_speed || best.data_throttle);
      bestData = best.data_throttle.map((v, i) => [bx[i], v]);
    }
    chart.setOption({ series: [
      { data: lap.data_throttle.map((v,i) => [x[i], v]) },
      { data: bestData },
    ]});
  }
});
