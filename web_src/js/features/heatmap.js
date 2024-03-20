import {createApp} from 'vue';
import ActivityHeatmap from '../components/ActivityHeatmap.vue';
import {translateMonth, translateDay} from '../utils.js';

export function initHeatmap() {
  const el = document.getElementById('user-heatmap');
  if (!el) return;

  try {
    const heatmap = {};
    for (const {contributions, timestamp} of JSON.parse(el.getAttribute('data-heatmap-data'))) {
      // Convert to user timezone and sum contributions by date
      const dateStr = new Date(timestamp * 1000).toDateString();
      heatmap[dateStr] = (heatmap[dateStr] || 0) + contributions;
    }

    const values = Object.keys(heatmap).map((v) => {
      return {date: new Date(v), count: heatmap[v]};
    });

    const locale = {
      months: new Array(12).fill().map((_, idx) => translateMonth(idx)),
      days: new Array(7).fill().map((_, idx) => translateDay(idx)),
      contributions_in_the_last_12_months: el.getAttribute('data-locale-total-contributions'),
      contributions_zero: el.getAttribute('data-locale-contributions-zero'),
      contributions_format: el.getAttribute('data-locale-contributions-format'),
      contributions_one: el.getAttribute('data-locale-contributions-one'),
      contributions_few: el.getAttribute('data-locale-contributions-few'),
      more: el.getAttribute('data-locale-more'),
      less: el.getAttribute('data-locale-less'),
    };

    const View = createApp(ActivityHeatmap, {values, locale});
    View.mount(el);
    el.classList.remove('is-loading');
  } catch (err) {
    console.error('Heatmap failed to load', err);
    el.textContent = 'Heatmap failed to load';
  }
}
