registerChart('gear', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['gear'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.gear'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value', min: 0, max: 8, interval: 1 },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#00bcd4' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['gear'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_gear) return;
    const x = xAxis(lap.data_speed || lap.data_gear);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_gear) {
      const bx = xAxis(best.data_speed || best.data_gear);
      bestData = best.data_gear.map((v, i) => [bx[i], v]);
    }
    chart.setOption({ series: [
      { data: lap.data_gear.map((v,i) => [x[i], v]) },
      { data: bestData },
    ]});
  }
});
