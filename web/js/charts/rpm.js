registerChart('rpm', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['rpm'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.rpm'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value' },
      series: [
        { type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#e91e63' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 30, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['rpm'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_rpm) return;
    const x = xAxis(lap.data_speed || lap.data_rpm);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_rpm) {
      const bx = xAxis(best.data_speed || best.data_rpm);
      bestData = best.data_rpm.map((v, i) => [bx[i], v]);
    }
    chart.setOption({ series: [
      { data: lap.data_rpm.map((v,i) => [x[i], v]) },
      { data: bestData },
    ]});
  }
});
