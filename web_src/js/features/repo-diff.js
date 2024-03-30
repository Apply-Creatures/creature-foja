import $ from 'jquery';
import {initCompReactionSelector} from './comp/ReactionSelector.js';
import {initRepoIssueContentHistory} from './repo-issue-content.js';
import {initDiffFileTree} from './repo-diff-filetree.js';
import {initDiffCommitSelect} from './repo-diff-commitselect.js';
import {validateTextareaNonEmpty} from './comp/ComboMarkdownEditor.js';
import {initViewedCheckboxListenerFor, countAndUpdateViewedFiles, initExpandAndCollapseFilesButton} from './pull-view-file.js';
import {initImageDiff} from './imagediff.js';
import {showErrorToast} from '../modules/toast.js';
import {submitEventSubmitter} from '../utils/dom.js';
import {POST, GET} from '../modules/fetch.js';

const {pageData, i18n} = window.config;

function initRepoDiffReviewButton() {
  const reviewBox = document.getElementById('review-box');
  if (!reviewBox) return;

  const $reviewBox = $(reviewBox);
  const counter = reviewBox.querySelector('.review-comments-counter');
  if (!counter) return;

  $(document).on('click', 'button[name="pending_review"]', (e) => {
    const $form = $(e.target).closest('form');
    // Watch for the form's submit event.
    $form.on('submit', () => {
      const num = parseInt(counter.getAttribute('data-pending-comment-number')) + 1 || 1;
      counter.setAttribute('data-pending-comment-number', num);
      counter.textContent = num;
      // Force the browser to reflow the DOM. This is to ensure that the browser replay the animation
      $reviewBox.removeClass('pulse');
      $reviewBox.width();
      $reviewBox.addClass('pulse');
    });
  });
}

function initRepoDiffFileViewToggle() {
  $('.file-view-toggle').on('click', function () {
    const $this = $(this);
    $this.parent().children().removeClass('active');
    $this.addClass('active');

    const $target = $($this.data('toggle-selector'));
    $target.parent().children().addClass('tw-hidden');
    $target.removeClass('tw-hidden');
  });
}

function initRepoDiffConversationForm() {
  $(document).on('submit', '.conversation-holder form', async (e) => {
    e.preventDefault();

    const $form = $(e.target);
    const textArea = e.target.querySelector('textarea');
    if (!validateTextareaNonEmpty(textArea)) {
      return;
    }

    if ($form.hasClass('is-loading')) return;
    try {
      $form.addClass('is-loading');
      const formData = new FormData($form[0]);

      // If the form is submitted by a button, append the button's name and value to the form data.
      // originalEvent can be undefined, such as an event that's caused by Ctrl+Enter, in that case
      // sent the event itself.
      const submitter = submitEventSubmitter(e.originalEvent ?? e);
      const isSubmittedByButton = (submitter?.nodeName === 'BUTTON') || (submitter?.nodeName === 'INPUT' && submitter.type === 'submit');
      if (isSubmittedByButton && submitter.name) {
        formData.append(submitter.name, submitter.value);
      }

      const response = await POST(e.target.getAttribute('action'), {data: formData});
      const $newConversationHolder = $(await response.text());
      const {path, side, idx} = $newConversationHolder.data();

      $form.closest('.conversation-holder').replaceWith($newConversationHolder);
      if ($form.closest('tr').data('line-type') === 'same') {
        $(`[data-path="${path}"] .add-code-comment[data-idx="${idx}"]`).addClass('tw-invisible');
      } else {
        $(`[data-path="${path}"] .add-code-comment[data-side="${side}"][data-idx="${idx}"]`).addClass('tw-invisible');
      }
      $newConversationHolder.find('.dropdown').dropdown();
      initCompReactionSelector($newConversationHolder);
    } catch { // here the caught error might be a jQuery AJAX error (thrown by await $.post), which is not good to use for error message handling
      console.error('error when submitting conversation', e);
      showErrorToast(i18n.network_error);
    } finally {
      $form.removeClass('is-loading');
    }
  });

  $(document).on('click', '.resolve-conversation', async function (e) {
    e.preventDefault();
    const comment_id = $(this).data('comment-id');
    const origin = $(this).data('origin');
    const action = $(this).data('action');
    const url = $(this).data('update-url');

    try {
      const response = await POST(url, {data: new URLSearchParams({origin, action, comment_id})});
      const data = await response.text();

      if ($(this).closest('.conversation-holder').length) {
        const $conversation = $(data);
        $(this).closest('.conversation-holder').replaceWith($conversation);
        $conversation.find('.dropdown').dropdown();
        initCompReactionSelector($conversation);
      } else {
        window.location.reload();
      }
    } catch (error) {
      console.error('Error:', error);
    }
  });
}

export function initRepoDiffConversationNav() {
  // Previous/Next code review conversation
  $(document).on('click', '.previous-conversation', (e) => {
    const $conversation = $(e.currentTarget).closest('.comment-code-cloud');
    const $conversations = $('.comment-code-cloud:not(.tw-hidden)');
    const index = $conversations.index($conversation);
    const previousIndex = index > 0 ? index - 1 : $conversations.length - 1;
    const $previousConversation = $conversations.eq(previousIndex);
    const anchor = $previousConversation.find('.comment').first()[0].getAttribute('id');
    window.location.href = `#${anchor}`;
  });
  $(document).on('click', '.next-conversation', (e) => {
    const $conversation = $(e.currentTarget).closest('.comment-code-cloud');
    const $conversations = $('.comment-code-cloud:not(.tw-hidden)');
    const index = $conversations.index($conversation);
    const nextIndex = index < $conversations.length - 1 ? index + 1 : 0;
    const $nextConversation = $conversations.eq(nextIndex);
    const anchor = $nextConversation.find('.comment').first()[0].getAttribute('id');
    window.location.href = `#${anchor}`;
  });
}

// Will be called when the show more (files) button has been pressed
function onShowMoreFiles() {
  initRepoIssueContentHistory();
  initViewedCheckboxListenerFor();
  countAndUpdateViewedFiles();
  initImageDiff();
}

export async function loadMoreFiles(url) {
  const $target = $('a#diff-show-more-files');
  if ($target.hasClass('disabled') || pageData.diffFileInfo.isLoadingNewData) {
    return;
  }

  pageData.diffFileInfo.isLoadingNewData = true;
  $target.addClass('disabled');

  try {
    const response = await GET(url);
    const resp = await response.text();
    const $resp = $(resp);
    // the response is a full HTML page, we need to extract the relevant contents:
    // 1. append the newly loaded file list items to the existing list
    $('#diff-incomplete').replaceWith($resp.find('#diff-file-boxes').children());
    // 2. re-execute the script to append the newly loaded items to the JS variables to refresh the DiffFileTree
    $('body').append($resp.find('script#diff-data-script'));

    onShowMoreFiles();
  } catch (error) {
    console.error('Error:', error);
    showErrorToast('An error occurred while loading more files.');
  } finally {
    $target.removeClass('disabled');
    pageData.diffFileInfo.isLoadingNewData = false;
  }
}

function initRepoDiffShowMore() {
  $(document).on('click', 'a#diff-show-more-files', (e) => {
    e.preventDefault();

    const linkLoadMore = e.target.getAttribute('data-href');
    loadMoreFiles(linkLoadMore);
  });

  $(document).on('click', 'a.diff-load-button', async (e) => {
    e.preventDefault();
    const $target = $(e.target);

    if ($target.hasClass('disabled')) {
      return;
    }

    $target.addClass('disabled');

    const url = $target.data('href');

    try {
      const response = await GET(url);
      const resp = await response.text();

      if (!resp) {
        return;
      }
      $target.parent().replaceWith($(resp).find('#diff-file-boxes .diff-file-body .file-body').children());
      onShowMoreFiles();
    } catch (error) {
      console.error('Error:', error);
    } finally {
      $target.removeClass('disabled');
    }
  });
}

export function initRepoDiffView() {
  initRepoDiffConversationForm();
  if (!$('#diff-file-list').length) return;
  initDiffFileTree();
  initDiffCommitSelect();
  initRepoDiffShowMore();
  initRepoDiffReviewButton();
  initRepoDiffFileViewToggle();
  initViewedCheckboxListenerFor();
  initExpandAndCollapseFilesButton();
}
