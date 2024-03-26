import $ from 'jquery';
import {minimatch} from 'minimatch';
import {createMonaco} from './codeeditor.js';
import {onInputDebounce, toggleElem} from '../utils/dom.js';
import {POST} from '../modules/fetch.js';

const {appSubUrl, csrfToken} = window.config;

export function initRepoSettingsCollaboration() {
  // Change collaborator access mode
  $('.page-content.repository .ui.dropdown.access-mode').each((_, el) => {
    const $dropdown = $(el);
    const $text = $dropdown.find('> .text');
    $dropdown.dropdown({
      async action(_text, value) {
        const lastValue = el.getAttribute('data-last-value');
        try {
          el.setAttribute('data-last-value', value);
          $dropdown.dropdown('hide');
          const data = new FormData();
          data.append('uid', el.getAttribute('data-uid'));
          data.append('mode', value);
          await POST(el.getAttribute('data-url'), {data});
        } catch {
          $text.text('(error)'); // prevent from misleading users when error occurs
          el.setAttribute('data-last-value', lastValue);
        }
      },
      onChange(_value, text, _$choice) {
        $text.text(text); // update the text when using keyboard navigating
      },
      onHide() {
        // set to the really selected value, defer to next tick to make sure `action` has finished its work because the calling order might be onHide -> action
        setTimeout(() => {
          const $item = $dropdown.dropdown('get item', el.getAttribute('data-last-value'));
          if ($item) {
            $dropdown.dropdown('set selected', el.getAttribute('data-last-value'));
          } else {
            $text.text('(none)'); // prevent from misleading users when the access mode is undefined
          }
        }, 0);
      },
    });
  });
}

export function initRepoSettingSearchTeamBox() {
  const searchTeamBox = document.getElementById('search-team-box');
  if (!searchTeamBox) return;

  $(searchTeamBox).search({
    minCharacters: 2,
    apiSettings: {
      url: `${appSubUrl}/org/${searchTeamBox.getAttribute('data-org-name')}/teams/-/search?q={query}`,
      headers: {'X-Csrf-Token': csrfToken},
      onResponse(response) {
        const items = [];
        $.each(response.data, (_i, item) => {
          items.push({
            title: item.name,
            description: `${item.permission} access`, // TODO: translate this string
          });
        });

        return {results: items};
      },
    },
    searchFields: ['name', 'description'],
    showNoResults: false,
  });
}

export function initRepoSettingGitHook() {
  if ($('.edit.githook').length === 0) return;
  const filename = document.querySelector('.hook-filename').textContent;
  const _promise = createMonaco($('#content')[0], filename, {language: 'shell'});
}

export function initRepoSettingBranches() {
  if (!$('.repository.settings.branches').length) return;
  $('.toggle-target-enabled').on('change', function () {
    const $target = $(this.getAttribute('data-target'));
    $target.toggleClass('disabled', !this.checked);
  });
  $('.toggle-target-disabled').on('change', function () {
    const $target = $(this.getAttribute('data-target'));
    if (this.checked) $target.addClass('disabled'); // only disable, do not auto enable
  });
  $('#dismiss_stale_approvals').on('change', function () {
    const $target = $('#ignore_stale_approvals_box');
    $target.toggleClass('disabled', this.checked);
  });

  // show the `Matched` mark for the status checks that match the pattern
  const markMatchedStatusChecks = () => {
    const patterns = (document.getElementById('status_check_contexts').value || '').split(/[\r\n]+/);
    const validPatterns = patterns.map((item) => item.trim()).filter(Boolean);
    const marks = document.getElementsByClassName('status-check-matched-mark');

    for (const el of marks) {
      let matched = false;
      const statusCheck = el.getAttribute('data-status-check');
      for (const pattern of validPatterns) {
        if (minimatch(statusCheck, pattern)) {
          matched = true;
          break;
        }
      }

      toggleElem(el, matched);
    }
  };
  markMatchedStatusChecks();
  document.getElementById('status_check_contexts').addEventListener('input', onInputDebounce(markMatchedStatusChecks));
}
