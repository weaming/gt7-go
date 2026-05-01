registerChart('raceline', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['raceline'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.raceline'), textStyle: { fontSize: 12 } },
      tooltip: { trigger: 'item' },
      xAxis: { type: 'value', scale: true },
      yAxis: { type: 'value', scale: true },
      series: [
        { name: i18n.t('misc.position'), type: 'line', showSymbol: false, lineStyle: { width: 2, color: '#4caf50' }, data: [] },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 1.5, color: '#ff9800', type: 'dashed' }, data: [] },
        { name: i18n.t('misc.brake_points'), type: 'scatter', symbol: 'pin', symbolSize: 20, color: '#f44336', data: [] },
      ],
      grid: { left: 10, right: 10, top: 30, bottom: 10 },
    });
  },
  update(laps, idx) {
    const chart = charts['raceline'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }, { data: [] }] }); return; }

    // Current/selected lap
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_position_x) return;

    // Find best completed lap (non-live, fastest time)
    let bestLap = null;
    let bestTime = Infinity;
    for (const l of laps) {
      if (l._is_live || !l.lap_finish_time || l.lap_finish_time <= 0) continue;
      if (l.lap_finish_time < bestTime) {
        bestTime = l.lap_finish_time;
        bestLap = l;
      }
    }

    const posData = lap.data_position_x.map((x, i) => [x, -lap.data_position_z[i]]);

    // Best lap reference line (dashed orange)
    let bestPosData = [];
    if (bestLap && bestLap !== lap && bestLap.data_position_x) {
      bestPosData = bestLap.data_position_x.map((x, i) => [x, -bestLap.data_position_z[i]]);
    }

    // Brake points for current lap
    const brakePoints = [];
    if (lap.data_braking) {
      for (let i = 0; i < lap.data_braking.length; i++) {
        if (lap.data_braking[i] > 0 && (i === 0 || lap.data_braking[i-1] === 0)) {
          if (i < lap.data_position_x.length && i < lap.data_position_z.length) {
            brakePoints.push({ value: [lap.data_position_x[i], -lap.data_position_z[i]] });
          }
        }
      }
    }

    chart.setOption({
      series: [
        { data: posData },
        { data: bestPosData },
        { data: brakePoints },
      ],
    });
  }
});
