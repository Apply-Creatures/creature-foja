import $ from 'jquery';
import {initCompColorPicker} from './ColorPicker.js';

function isExclusiveScopeName(name) {
  return /.*[^/]\/[^/].*/.test(name);
}

function updateExclusiveLabelEdit(form) {
  const $nameInput = $(`${form} .label-name-input`);
  const $exclusiveField = $(`${form} .label-exclusive-input-field`);
  const $exclusiveCheckbox = $(`${form} .label-exclusive-input`);
  const $exclusiveWarning = $(`${form} .label-exclusive-warning`);

  if (isExclusiveScopeName($nameInput.val())) {
    $exclusiveField.removeClass('muted');
    $exclusiveField.removeAttr('aria-disabled');
    if ($exclusiveCheckbox[0].checked && $exclusiveCheckbox.data('exclusive-warn')) {
      $exclusiveWarning.removeClass('gt-hidden');
    } else {
      $exclusiveWarning.addClass('gt-hidden');
    }
  } else {
    $exclusiveField.addClass('muted');
    $exclusiveField.attr('aria-disabled', 'true');
    $exclusiveWarning.addClass('gt-hidden');
  }
}

export function initCompLabelEdit(selector) {
  if (!$(selector).length) return;
  initCompColorPicker();

  // Create label
  $('.new-label.button').on('click', () => {
    updateExclusiveLabelEdit('.new-label');
    $('.new-label.modal').modal({
      onApprove() {
        $('.new-label.form').trigger('submit');
      },
    }).modal('show');
    return false;
  });

  // Edit label
  $('.edit-label-button').on('click', function () {
    $('#label-modal-id').val($(this).data('id'));

    const $nameInput = $('.edit-label .label-name-input');
    $nameInput.val($(this).data('title'));

    const $isArchivedCheckbox = $('.edit-label .label-is-archived-input');
    $isArchivedCheckbox[0].checked = this.hasAttribute('data-is-archived');

    const $exclusiveCheckbox = $('.edit-label .label-exclusive-input');
    $exclusiveCheckbox[0].checked = this.hasAttribute('data-exclusive');
    // Warn when label was previously not exclusive and used in issues
    $exclusiveCheckbox.data('exclusive-warn',
      $(this).data('num-issues') > 0 &&
      (!this.hasAttribute('data-exclusive') || !isExclusiveScopeName($nameInput.val())));
    updateExclusiveLabelEdit('.edit-label');

    $('.edit-label .label-desc-input').val(this.getAttribute('data-description'));
    $('.edit-label .color-picker').minicolors('value', this.getAttribute('data-color'));

    $('.edit-label.modal').modal({
      onApprove() {
        $('.edit-label.form').trigger('submit');
      },
    }).modal('show');
    return false;
  });

  $('.new-label .label-name-input').on('input', () => {
    updateExclusiveLabelEdit('.new-label');
  });
  $('.new-label .label-exclusive-input').on('change', () => {
    updateExclusiveLabelEdit('.new-label');
  });
  $('.edit-label .label-name-input').on('input', () => {
    updateExclusiveLabelEdit('.edit-label');
  });
  $('.edit-label .label-exclusive-input').on('change', () => {
    updateExclusiveLabelEdit('.edit-label');
  });
}
