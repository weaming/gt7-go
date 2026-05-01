registerChart('raceline', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['raceline'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.raceline'), textStyle: { fontSize: 14, color: '#eee' } },
      tooltip: { trigger: 'item', formatter: (p) => p.seriesName },
      xAxis: { type: 'value', scale: true, splitLine: { show: false }, axisLabel: { show: false }, axisTick: { show: false } },
      yAxis: { type: 'value', scale: true, splitLine: { show: false }, axisLabel: { show: false }, axisTick: { show: false } },
      legend: { textStyle: { color: '#aaa' }, top: 28, itemWidth: 14, itemHeight: 3 },
      series: [
        { name: i18n.t('chart.throttle'), type: 'line', showSymbol: false, lineStyle: { width: 3, color: '#4caf50' }, data: [], z: 2 },
        { name: i18n.t('chart.braking'), type: 'line', showSymbol: false, lineStyle: { width: 3, color: '#f44336' }, data: [], z: 2 },
        { name: i18n.t('chart.coasting'), type: 'line', showSymbol: false, lineStyle: { width: 2, color: '#42a5f5', opacity: 0.6 }, data: [], z: 1 },
        { name: i18n.t('misc.best_lap'), type: 'line', showSymbol: false, lineStyle: { width: 2, color: '#ff9800', type: 'dashed' }, data: [], z: 3 },
        { name: i18n.t('misc.brake_points'), type: 'scatter', symbol: 'diamond', symbolSize: 14, color: '#ff5722', itemStyle: { borderColor: '#fff', borderWidth: 1 }, data: [], z: 4 },
      ],
      grid: { left: 5, right: 5, top: 55, bottom: 5 },
    });
  },
  update(laps, idx) {
    const chart = charts['raceline'];
    if (!chart) return;
    if (laps.length === 0) { chart.setOption({ series: [{ data: [] }, { data: [] }, { data: [] }, { data: [] }, { data: [] }] }); return; }

    const lap = laps[idx] || laps[0];
    if (!lap || !lap.data_position_x) return;

    // Best completed lap for reference
    let bestLap = null;
    let bestTime = Infinity;
    for (const l of laps) {
      if (!isRankableLap(l)) continue;
      if (l.lap_finish_time < bestTime) { bestTime = l.lap_finish_time; bestLap = l; }
    }

    const n = lap.data_position_x.length;
    const posZ = lap.data_position_z || [];
    const thr = lap.data_throttle || [];
    const brk = lap.data_braking || [];

    // Build color-coded track segments
    const thrData = [];
    const brkData = [];
    const cstData = [];
    for (let i = 0; i < n; i++) {
      const z = -posZ[i];
      const p = [lap.data_position_x[i], z];
      const t = thr[i] || 0;
      const b = brk[i] || 0;
      if (t > 0 && b === 0) {
        thrData.push(p); brkData.push(null); cstData.push(null);
      } else if (b > 0) {
        thrData.push(null); brkData.push(p); cstData.push(null);
      } else {
        thrData.push(null); brkData.push(null); cstData.push(p);
      }
    }

    // Best lap trace (dashed orange)
    let bestData = [];
    if (bestLap && bestLap !== lap && bestLap.data_position_x) {
      bestData = bestLap.data_position_x.map((x, i) => [x, -bestLap.data_position_z[i]]);
    }

    // Brake points
    const brakePoints = [];
    if (brk.length > 0) {
      for (let i = 1; i < brk.length && i < lap.data_position_x.length; i++) {
        if (brk[i] > 0 && brk[i - 1] === 0) {
          brakePoints.push({ value: [lap.data_position_x[i], -posZ[i]] });
        }
      }
    }

    chart.setOption({
      series: [
        { data: thrData },
        { data: brkData },
        { data: cstData },
        { data: bestData },
        { data: brakePoints },
      ],
    });
  }
});
