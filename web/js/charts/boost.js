registerChart('boost', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['boost'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.boost'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value' },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#2196f3' }, areaStyle: { opacity: 0.2 }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['boost'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_boost) return;
    const x = xAxis(lap.data_speed || lap.data_boost);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_boost) {
      const bx = xAxis(best.data_speed || best.data_boost);
      bestData = best.data_boost.map((v, i) => [bx[i], v]);
    }
    chart.setOption({ series: [
      { data: lap.data_boost.map((v,i) => [x[i], v < 0 ? 0 : v]) },
      { data: bestData.map(([xi,v]) => [xi, v < 0 ? 0 : v]) },
    ]});
  }
});
