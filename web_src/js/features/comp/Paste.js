import {POST} from '../../modules/fetch.js';
import {getPastedContent, replaceTextareaSelection} from '../../utils/dom.js';
import {isUrl} from '../../utils/url.js';

async function uploadFile(file, uploadUrl) {
  const formData = new FormData();
  formData.append('file', file, file.name);

  const res = await POST(uploadUrl, {data: formData});
  return await res.json();
}

function triggerEditorContentChanged(target) {
  target.dispatchEvent(new CustomEvent('ce-editor-content-changed', {bubbles: true}));
}

class TextareaEditor {
  constructor(editor) {
    this.editor = editor;
  }

  insertPlaceholder(value) {
    const editor = this.editor;
    const startPos = editor.selectionStart;
    const endPos = editor.selectionEnd;
    editor.value = editor.value.substring(0, startPos) + value + editor.value.substring(endPos);
    editor.selectionStart = startPos;
    editor.selectionEnd = startPos + value.length;
    editor.focus();
    triggerEditorContentChanged(editor);
  }

  replacePlaceholder(oldVal, newVal) {
    const editor = this.editor;
    const startPos = editor.selectionStart;
    const endPos = editor.selectionEnd;
    if (editor.value.substring(startPos, endPos) === oldVal) {
      editor.value = editor.value.substring(0, startPos) + newVal + editor.value.substring(endPos);
      editor.selectionEnd = startPos + newVal.length;
    } else {
      editor.value = editor.value.replace(oldVal, newVal);
      editor.selectionEnd -= oldVal.length;
      editor.selectionEnd += newVal.length;
    }
    editor.selectionStart = editor.selectionEnd;
    editor.focus();
    triggerEditorContentChanged(editor);
  }
}

class CodeMirrorEditor {
  constructor(editor) {
    this.editor = editor;
  }

  insertPlaceholder(value) {
    const editor = this.editor;
    const startPoint = editor.getCursor('start');
    const endPoint = editor.getCursor('end');
    editor.replaceSelection(value);
    endPoint.ch = startPoint.ch + value.length;
    editor.setSelection(startPoint, endPoint);
    editor.focus();
    triggerEditorContentChanged(editor.getTextArea());
  }

  replacePlaceholder(oldVal, newVal) {
    const editor = this.editor;
    const endPoint = editor.getCursor('end');
    if (editor.getSelection() === oldVal) {
      editor.replaceSelection(newVal);
    } else {
      editor.setValue(editor.getValue().replace(oldVal, newVal));
    }
    endPoint.ch -= oldVal.length;
    endPoint.ch += newVal.length;
    editor.setSelection(endPoint, endPoint);
    editor.focus();
    triggerEditorContentChanged(editor.getTextArea());
  }
}

async function handleClipboardImages(editor, dropzone, images, e) {
  const uploadUrl = dropzone.getAttribute('data-upload-url');
  const filesContainer = dropzone.querySelector('.files');

  if (!dropzone || !uploadUrl || !filesContainer || !images.length) return;

  e.preventDefault();
  e.stopPropagation();

  for (const img of images) {
    const name = img.name.slice(0, img.name.lastIndexOf('.'));

    const placeholder = `![${name}](uploading ...)`;
    editor.insertPlaceholder(placeholder);

    const {uuid} = await uploadFile(img, uploadUrl);

    const url = `/attachments/${uuid}`;
    const text = `![${name}](${url})`;
    editor.replacePlaceholder(placeholder, text);

    const input = document.createElement('input');
    input.setAttribute('name', 'files');
    input.setAttribute('type', 'hidden');
    input.setAttribute('id', uuid);
    input.value = uuid;
    filesContainer.append(input);
  }
}

function handleClipboardText(textarea, text, e) {
  // when pasting links over selected text, turn it into [text](link), except when shift key is held
  const {value, selectionStart, selectionEnd, _shiftDown} = textarea;
  if (_shiftDown) return;
  const selectedText = value.substring(selectionStart, selectionEnd);
  const trimmedText = text.trim();
  if (selectedText && isUrl(trimmedText)) {
    e.stopPropagation();
    e.preventDefault();
    replaceTextareaSelection(textarea, `[${selectedText}](${trimmedText})`);
  }
}

export function initEasyMDEPaste(easyMDE, dropzone) {
  easyMDE.codemirror.on('paste', (_, e) => {
    const {images} = getPastedContent(e);
    if (images.length) {
      handleClipboardImages(new CodeMirrorEditor(easyMDE.codemirror), dropzone, images, e);
    }
  });
}

export function initTextareaPaste(textarea, dropzone) {
  textarea.addEventListener('paste', (e) => {
    const {images, text} = getPastedContent(e);
    if (images.length) {
      handleClipboardImages(new TextareaEditor(textarea), dropzone, images, e);
    } else if (text) {
      handleClipboardText(textarea, text, e);
    }
  });
}
