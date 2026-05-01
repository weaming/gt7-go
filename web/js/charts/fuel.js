registerChart('fuel', {
  init(el, charts) {
    const chart = echarts.init(el, 'dark');
    charts['fuel'] = chart;
    chart.setOption({
      title: { text: i18n.t('chart.fuel'), textStyle: { fontSize: 12 } },
      tooltip: {
        trigger: 'axis',
        formatter(params) {
          let s = params[0].axisValueLabel + '<br/>';
          params.forEach(p => {
            const v = p.seriesName === i18n.t('misc.time_remaining')
              ? p.value + 's' : p.value;
            s += p.marker + ' ' + p.seriesName + ': ' + v + '<br/>';
          });
          return s;
        },
      },
      legend: { textStyle: { color: '#aaa' }, top: 25 },
      xAxis: { type: 'category', data: [] },
      yAxis: [
        { type: 'value', name: i18n.t('misc.laps') },
        { type: 'value', name: i18n.t('misc.time_s') },
      ],
      series: [
        { name: i18n.t('misc.laps_remaining'), type: 'bar', data: [] },
        { name: i18n.t('misc.time_remaining'), type: 'line', yAxisIndex: 1, data: [] },
      ],
      grid: { left: 50, right: 50, top: 55, bottom: 20 },
    });
  },
  update(laps, idx) {
    const chart = charts['fuel'];
    if (!chart) return;
    const lap = laps[idx] || laps[0];
    if (!lap || !lap.fuel_at_start) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    if (!lap.fuel_consumed) { chart.setOption({ series: [{ data: [] }, { data: [] }] }); return; }
    const maps = getFuelMaps(lap);
    const labels = maps.map(m => m.mixture_setting > 0 ? '+' + m.mixture_setting : String(m.mixture_setting));
    chart.setOption({
      xAxis: { data: labels },
      series: [
        { data: maps.map(m => m.laps_remaining) },
        { data: maps.map(m => m.time_remaining) },
      ],
    });
  }
});

function getFuelMaps(lap) {
  const consumed = lap.fuel_consumed || 0;
  const lapTime = lap.lap_finish_time || 0;
  const results = [];
  for (let i = -5; i <= 5; i++) {
    const powerPct = (100 - i * 4) / 100;
    const consPct = (100 - i * 8) / 100;
    const adjCons = consumed * consPct;
    const lapsOnFuel = adjCons > 0 ? (lap.fuel_at_end || 0) / adjCons : 0;
    const timeRemaining = lapsOnFuel * lapTime;
    const timeDiff = -(powerPct - 1) * lapTime;
    results.push({
      mixture_setting: i, power_percent: powerPct, consumption_percent: consPct,
      fuel_consumed_per_lap: r1(adjCons),
      laps_remaining: r1(lapsOnFuel),
      time_remaining: Math.round(timeRemaining / 1000),
      lap_time_diff: Math.round(timeDiff),
      lap_time_expected: Math.round(lapTime + timeDiff),
    });
  }
  return results;
}

function r1(v) { return Math.round(v * 10) / 10; }
