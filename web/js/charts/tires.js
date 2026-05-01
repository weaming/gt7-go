registerChart('tires', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['tires'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.tires'), textStyle: { fontSize: 12 } },
      tooltip: {
        trigger: 'axis',
        formatter: (params) => {
          const v = (params[0].value[1] * 100).toFixed(1);
          return `${v}%`;
        }
      },
      legend: { data: [i18n.t('chart.tires_avg'), i18n.t('chart.tires_max')], top: 20, textStyle: { fontSize: 10 } },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'value', axisLabel: { formatter: (v) => (v * 100).toFixed(0) + '%' } },
      series: [
        { name: i18n.t('chart.tires_avg'), type: 'line', showSymbol: false, lineStyle: { width: 1.5, color: '#00d5ff' }, data: [] },
        { name: i18n.t('chart.tires_max'), type: 'line', showSymbol: false, lineStyle: { width: 1.5, color: '#e94560' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1, color: '#ff9800', type: 'dashed' }, data: [] },
      ],
      grid: { left: 50, right: 10, top: 50, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['tires'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }, { data: [] }] }); return; }
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_tires) return;
    const x = xAxis(lap.data_speed || lap.data_tires);
    const best = getBestLap(laps);
    let bestData = [];
    if (best && best !== lap && best.data_tires) {
      const bx = xAxis(best.data_speed || best.data_tires);
      bestData = best.data_tires.map((v, i) => [bx[i], v]);
    }
    let maxData = [];
    if (lap.data_tire_slip_max) {
      maxData = lap.data_tire_slip_max.map((v, i) => [x[i], v]);
    }
    chart.setOption({ series: [
      { name: i18n.t('chart.tires_avg'), data: lap.data_tires.map((v,i) => [x[i], v]) },
      { name: i18n.t('chart.tires_max'), data: maxData },
      { name: i18n.t('misc.best_lap'), data: bestData },
    ]});
  }
});
