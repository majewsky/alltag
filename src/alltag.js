const all = selector => document.querySelectorAll(selector);

////////////////////////////////////////////////////////////////////////////////
// helper for initializing <select> state

for (const elem of all('select[data-initial-value]')) {
  if (elem.value == '' && elem.dataset.initialValue != '') {
    elem.value = elem.dataset.initialValue;
  }
}
