import {hideElem, showElem} from '../utils/dom.js';

function onPronounsDropdownUpdate() {
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsInput = document.querySelector('#pronouns-dropdown input');
  const isCustom = !(
    pronounsInput.value === 'he/him' ||
    pronounsInput.value === 'she/her' ||
    pronounsInput.value === 'they/them' ||
    pronounsInput.value === 'it/its'
  );
  if (isCustom) {
    pronounsCustom.value = pronounsInput.value;
    pronounsCustom.style.display = '';
  } else {
    pronounsCustom.style.display = 'none';
  }
}
function onPronounsCustomUpdate() {
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsInput = document.querySelector('#pronouns-dropdown input');
  pronounsInput.value = pronounsCustom.value;
}

export function initUserSettings() {
  if (!document.querySelectorAll('.user.settings.profile').length) return;

  const usernameInput = document.getElementById('username');
  if (!usernameInput) return;
  usernameInput.addEventListener('input', function () {
    const prompt = document.getElementById('name-change-prompt');
    const promptRedirect = document.getElementById('name-change-redirect-prompt');
    if (this.value.toLowerCase() !== this.getAttribute('data-name').toLowerCase()) {
      showElem(prompt);
      showElem(promptRedirect);
    } else {
      hideElem(prompt);
      hideElem(promptRedirect);
    }
  });

  const pronounsDropdown = document.getElementById('pronouns-dropdown');
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsInput = pronounsDropdown.querySelector('input');
  pronounsCustom.removeAttribute('name');
  pronounsDropdown.style.display = '';
  onPronounsDropdownUpdate();
  pronounsInput.addEventListener('change', onPronounsDropdownUpdate);
  pronounsCustom.addEventListener('input', onPronounsCustomUpdate);
}
