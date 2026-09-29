const form = document.querySelector('#form');
const status = document.querySelector('#status');
async function update(options) {
  try {
    const response = await fetch('api/notes', options);
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    document.querySelector('#notes').replaceChildren(...data.notes.map(text => {
      const item = document.createElement('li'); item.textContent = text; return item;
    }));
    status.textContent = data.notes.length ? `${data.notes.length} ${data.notes.length === 1 ? "note" : "notes"}` : 'No notes yet.';
    return true;
  } catch (error) { status.textContent = error.message; return false; }
}
form.addEventListener('submit', async event => {
  event.preventDefault();
  const button = form.querySelector('button'); button.disabled = true;
  const saved = await update({method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text: form.elements.note.value})});
  if (saved) form.reset();
  button.disabled = false;
});
document.querySelector('#refresh').addEventListener('click', () => update());
update();
