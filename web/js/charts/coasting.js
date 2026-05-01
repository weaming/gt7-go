registerChart('coasting', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['coasting'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.coasting'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value', min: 0, max: 1 },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx, liveLap) {
    const chart = charts['coasting'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = liveLap || laps[idx] || laps[0];
    if (!lap || !lap.data_coasting) return;
    const x = xAxis(lap.data_speed || lap.data_coasting);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_coasting) {
      const bx = xAxis(best.data_speed || best.data_coasting);
      bestData = best.data_coasting.map((v, i) => [bx[i], v]);
    }
    chart.setOption({ series: [
      { data: lap.data_coasting.map((v,i) => [x[i], v]) },
      { data: bestData },
    ]});
  }
});
