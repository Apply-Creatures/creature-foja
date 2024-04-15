import $ from 'jquery';
import {hideElem, showElem} from '../utils/dom.js';
import {GET} from '../modules/fetch.js';

export function initRepoGraphGit() {
  const graphContainer = document.getElementById('git-graph-container');
  if (!graphContainer) return;

  document.getElementById('flow-color-monochrome')?.addEventListener('click', () => {
    document.getElementById('flow-color-monochrome').classList.add('active');
    document.getElementById('flow-color-colored')?.classList.remove('active');
    graphContainer.classList.remove('colored');
    graphContainer.classList.add('monochrome');
    const params = new URLSearchParams(window.location.search);
    params.set('mode', 'monochrome');
    const queryString = params.toString();
    if (queryString) {
      window.history.replaceState({}, '', `?${queryString}`);
    } else {
      window.history.replaceState({}, '', window.location.pathname);
    }
    for (const link of document.querySelectorAll('.pagination a')) {
      const href = link.getAttribute('href');
      if (!href) continue;
      const url = new URL(href, window.location);
      const params = url.searchParams;
      params.set('mode', 'monochrome');
      url.search = `?${params.toString()}`;
      link.setAttribute('href', url.href);
    }
  });

  document.getElementById('flow-color-colored')?.addEventListener('click', () => {
    document.getElementById('flow-color-colored').classList.add('active');
    document.getElementById('flow-color-monochrome')?.classList.remove('active');
    graphContainer.classList.add('colored');
    graphContainer.classList.remove('monochrome');
    for (const link of document.querySelectorAll('.pagination a')) {
      const href = link.getAttribute('href');
      if (!href) continue;
      const url = new URL(href, window.location);
      const params = url.searchParams;
      params.delete('mode');
      url.search = `?${params.toString()}`;
      link.setAttribute('href', url.href);
    }
    const params = new URLSearchParams(window.location.search);
    params.delete('mode');
    const queryString = params.toString();
    if (queryString) {
      window.history.replaceState({}, '', `?${queryString}`);
    } else {
      window.history.replaceState({}, '', window.location.pathname);
    }
  });
  const url = new URL(window.location);
  const params = url.searchParams;
  const updateGraph = () => {
    const queryString = params.toString();
    const ajaxUrl = new URL(url);
    ajaxUrl.searchParams.set('div-only', 'true');
    window.history.replaceState({}, '', queryString ? `?${queryString}` : window.location.pathname);
    document.getElementById('pagination').innerHTML = '';
    hideElem('#rel-container');
    hideElem('#rev-container');
    showElem('#loading-indicator');
    (async () => {
      const response = await GET(String(ajaxUrl));
      const html = await response.text();
      const div = document.createElement('div');
      div.innerHTML = html;
      document.getElementById('pagination').innerHTML = div.getElementById('pagination').innerHTML;
      document.getElementById('rel-container').innerHTML = div.getElementById('rel-container').innerHTML;
      document.getElementById('rev-container').innerHTML = div.getElementById('rev-container').innerHTML;
      hideElem('#loading-indicator');
      showElem('#rel-container');
      showElem('#rev-container');
    })();
  };
  const dropdownSelected = params.getAll('branch');
  if (params.has('hide-pr-refs') && params.get('hide-pr-refs') === 'true') {
    dropdownSelected.splice(0, 0, '...flow-hide-pr-refs');
  }

  const flowSelectRefsDropdown = document.getElementById('flow-select-refs-dropdown');
  $(flowSelectRefsDropdown).dropdown('set selected', dropdownSelected);
  $(flowSelectRefsDropdown).dropdown({
    clearable: true,
    fullTextSeach: 'exact',
    onRemove(toRemove) {
      if (toRemove === '...flow-hide-pr-refs') {
        params.delete('hide-pr-refs');
      } else {
        const branches = params.getAll('branch');
        params.delete('branch');
        for (const branch of branches) {
          if (branch !== toRemove) {
            params.append('branch', branch);
          }
        }
      }
      updateGraph();
    },
    onAdd(toAdd) {
      if (toAdd === '...flow-hide-pr-refs') {
        params.set('hide-pr-refs', true);
      } else {
        params.append('branch', toAdd);
      }
      updateGraph();
    },
  });

  graphContainer.addEventListener('mouseenter', (e) => {
    if (e.target.matches('#rev-list li')) {
      const flow = e.target.getAttribute('data-flow');
      if (flow === '0') return;
      document.getElementById(`flow-${flow}`)?.classList.add('highlight');
      e.target.classList.add('hover');
      for (const item of document.querySelectorAll(`#rev-list li[data-flow='${flow}']`)) {
        item.classList.add('highlight');
      }
    } else if (e.target.matches('#rel-container .flow-group')) {
      e.target.classList.add('highlight');
      const flow = e.target.getAttribute('data-flow');
      for (const item of document.querySelectorAll(`#rev-list li[data-flow='${flow}']`)) {
        item.classList.add('highlight');
      }
    } else if (e.target.matches('#rel-container .flow-commit')) {
      const rev = e.target.getAttribute('data-rev');
      document.querySelector(`#rev-list li#commit-${rev}`)?.classList.add('hover');
    }
  });

  graphContainer.addEventListener('mouseleave', (e) => {
    if (e.target.matches('#rev-list li')) {
      const flow = e.target.getAttribute('data-flow');
      if (flow === '0') return;
      document.getElementById(`flow-${flow}`)?.classList.remove('highlight');
      e.target.classList.remove('hover');
      for (const item of document.querySelectorAll(`#rev-list li[data-flow='${flow}']`)) {
        item.classList.remove('highlight');
      }
    } else if (e.target.matches('#rel-container .flow-group')) {
      e.target.classList.remove('highlight');
      const flow = e.target.getAttribute('data-flow');
      for (const item of document.querySelectorAll(`#rev-list li[data-flow='${flow}']`)) {
        item.classList.remove('highlight');
      }
    } else if (e.target.matches('#rel-container .flow-commit')) {
      const rev = e.target.getAttribute('data-rev');
      document.querySelector(`#rev-list li#commit-${rev}`)?.classList.remove('hover');
    }
  });
}
