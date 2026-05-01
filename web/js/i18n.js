const i18n = {
  lang: 'zh',
  translations: {
    'tab.driver': { zh: '驾驶仪表', en: 'Driver Dash' },
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
    'chart.raceline': { zh: '赛车线（俯视）', en: 'Racing Line (Top View)' },
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
    'table.tb': { zh: '油刹重叠%', en: 'Overlap%' },
    'table.track': { zh: '赛道', en: 'Track' },
    'table.car': { zh: '车辆', en: 'Car' },
    'table.note': { zh: '备注', en: 'Notes' },
    'table.action': { zh: '操作', en: 'Action' },
    'table.pit': { zh: '进站', en: 'PIT' },
    'table.fastest': { zh: '最快', en: 'Best' },
    'table.slowest': { zh: '最慢', en: 'Slow' },
    'table.ref': { zh: '参考', en: 'Ref' },

    'table.target': { zh: '目标', en: 'Target' },

    'status.live': { zh: '实时', en: 'LIVE' },
    'status.disconnected': { zh: '未连接', en: 'Disconnected' },
    'status.connected': { zh: '已连接', en: 'Connected' },
    'status.backend': { zh: '网页', en: 'Web' },
    'status.ps5': { zh: 'PS5', en: 'PS5' },
    'status.ps5_live': { zh: '上游已连接', en: 'Upstream connected' },
    'status.ps5_waiting': { zh: '等待上游', en: 'Waiting upstream' },
    'status.finished': { zh: '已完赛', en: 'FINISHED' },

    'driver.speed': { zh: '速度', en: 'Speed' },
    'driver.gear': { zh: '档位', en: 'Gear' },
    'driver.rpm': { zh: '转速', en: 'RPM' },
    'driver.throttle': { zh: '油门', en: 'Throttle' },
    'driver.brake': { zh: '刹车', en: 'Brake' },
    'driver.current_lap': { zh: '当前圈', en: 'Current Lap' },
    'driver.live_time': { zh: '本圈计时', en: 'Lap Time' },
    'driver.live_diff': { zh: '参考差值', en: 'Reference Diff' },
    'driver.last_lap': { zh: '上一圈', en: 'Last Lap' },
    'driver.best_lap': { zh: '最佳圈', en: 'Best Lap' },
    'driver.fuel': { zh: '燃油', en: 'Fuel' },
    'driver.boost': { zh: '增压', en: 'Boost' },
    'driver.tire_slip': { zh: '轮胎滑移', en: 'Tire Slip' },
    'driver.water_temp': { zh: '水温', en: 'Water Temp' },
    'driver.oil_temp': { zh: '油温', en: 'Oil Temp' },
    'driver.suggested_gear': { zh: '建议', en: 'Suggested' },
    'driver.no_suggested_gear': { zh: '无建议', en: 'No hint' },

    'misc.theoretical_best': { zh: '理论最佳', en: 'Theoretical Best' },
    'misc.best_lap': { zh: '最佳圈', en: 'Best Lap' },
    'misc.consistency': { zh: '一致性标准差', en: 'Consistency StdDev' },
    'misc.time_lost': { zh: '可提升', en: 'Time Lost' },
    'misc.potential': { zh: '理论可提升', en: 'Potential Gain' },
    'misc.segment': { zh: '弯', en: 'Turn' },
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
    'misc.record_replay': { zh: '回放录制', en: 'Replay Record' },
    'misc.load': { zh: '加载', en: 'Load' },
    'misc.save_all': { zh: '保存全部', en: 'Save All' },
    'misc.clear': { zh: '清空', en: 'Clear' },
    'misc.delete': { zh: '删除', en: 'Delete' },
    'misc.confirm_delete_file': { zh: '确认删除选中的保存文件？不会清空当前 laps.json。', en: 'Delete the selected saved file? Current laps.json will not be cleared.' },
    'misc.confirm_clear': { zh: '确认清空当前圈和未保存历史圈数据？已保存的 JSON 文件不会删除。', en: 'Clear current lap and unsaved history laps data? Saved JSON files will not be deleted.' },

    'lang.zh': { zh: '中文', en: '中文' },
    'lang.en': { zh: 'EN', en: 'EN' },

    'shift.title': { zh: '换挡点分析', en: 'Shift Analysis' },
    'shift.gear': { zh: '档位', en: 'Gear' },
    'shift.pre_rpm': { zh: '换挡前 RPM', en: 'Pre-shift RPM' },
    'shift.post_rpm': { zh: '换挡后 RPM', en: 'Post-shift RPM' },
    'shift.drop': { zh: '落差', en: 'Drop' },
    'shift.drop_pct': { zh: '落差比', en: 'Drop %' },
    'shift.avg_rpm': { zh: '平均换挡转速', en: 'Avg Shift RPM' },
    'shift.no_data': { zh: '无换挡数据（需先完成一圈）', en: 'No shift data (complete a lap first)' },
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
