const i18n = {
  lang: 'zh',
  translations: {
    'tab.analysis': { zh: '圈速分析', en: 'Lap Analysis' },
    'tab.raceline': { zh: '赛车线', en: 'Racing Line' },
    'tab.race': { zh: '比赛', en: 'Race' },
    'tab.corner': { zh: '弯道分析', en: 'Corner Analysis' },

    'chart.speed': { zh: '速度 (km/h)', en: 'Speed (km/h)' },
    'chart.timediff': { zh: '时间差 (ms)', en: 'Time Difference (ms)' },
    'chart.throttle': { zh: '油门 (%)', en: 'Throttle (%)' },
    'chart.braking': { zh: '刹车 (%)', en: 'Brake (%)' },
    'chart.coasting': { zh: '滑行', en: 'Coasting' },
    'chart.yawrate': { zh: '横摆角速度 (rad/s)', en: 'Yaw Rate (rad/s)' },
    'chart.gear': { zh: '档位', en: 'Gear' },
    'chart.rpm': { zh: '转速', en: 'RPM' },
    'chart.boost': { zh: '增压值 (bar)', en: 'Boost (bar)' },
    'chart.tires': { zh: '轮胎滑移率', en: 'Tire Slip Ratio' },
    'chart.variance': { zh: '速度方差 (圈间)', en: 'Speed Variance (across laps)' },
    'chart.raceline': { zh: '赛车线', en: 'Racing Line' },
    'chart.fuel': { zh: '燃油策略', en: 'Fuel Strategy' },
    'chart.corner': { zh: '弯道分析', en: 'Corner Analysis' },

    'table.num': { zh: '#', en: '#' },
    'table.time': { zh: '时间', en: 'Time' },
    'table.diff': { zh: '差值', en: 'Diff' },
    'table.fuel': { zh: '燃油', en: 'Fuel' },
    'table.thr': { zh: '油%', en: 'Thr%' },
    'table.brk': { zh: '刹%', en: 'Brk%' },
    'table.cst': { zh: '滑%', en: 'Cst%' },
    'table.spin': { zh: '滑移%', en: 'Spin%' },
    'table.tb': { zh: '油刹%', en: 'TB%' },
    'table.car': { zh: '车辆', en: 'Car' },

    'status.live': { zh: '实时', en: 'LIVE' },
    'status.disconnected': { zh: '未连接', en: 'Disconnected' },

    'misc.theoretical_best': { zh: '理论最佳', en: 'Theoretical Best' },
    'misc.best_lap': { zh: '最佳圈', en: 'Best Lap' },
    'misc.consistency': { zh: '一致性标准差', en: 'Consistency StdDev' },
    'misc.laps_remaining': { zh: '剩余圈数', en: 'Laps Remaining' },
    'misc.time_remaining': { zh: '剩余时间 (s)', en: 'Time Remaining (s)' },
    'misc.position': { zh: '位置', en: 'Position' },
    'misc.brake_points': { zh: '刹车点', en: 'Brake Points' },
    'misc.distance_km': { zh: '距离 (km)', en: 'Distance (km)' },
    'misc.time_ms': { zh: '时间 (ms)', en: 'Time (ms)' },
    'misc.stddev': { zh: '标准差', en: 'StdDev' },
    'misc.bar': { zh: '巴', en: 'bar' },
    'misc.laps': { zh: '圈数', en: 'Laps' },
    'misc.time_s': { zh: '时间 (s)', en: 'Time (s)' },

    'lang.zh': { zh: '中文', en: '中文' },
    'lang.en': { zh: 'EN', en: 'EN' },
  },

  t(key) {
    const entry = this.translations[key];
    return entry ? (entry[this.lang] || key) : key;
  },

  toggle() {
    this.lang = this.lang === 'zh' ? 'en' : 'zh';
    localStorage.setItem('gt7_lang', this.lang);
    location.reload();
  },

  apply() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
      el.textContent = this.t(el.dataset.i18n);
    });
  },

  init() {
    const saved = localStorage.getItem('gt7_lang');
    if (saved) {
      this.lang = saved;
    } else {
      this.lang = navigator.language && navigator.language.startsWith('zh') ? 'zh' : 'en';
    }
  },
};

i18n.init();
