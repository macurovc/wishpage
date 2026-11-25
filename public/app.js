// Progressive enhancement. Without JavaScript the people rail navigates
// normally (?open=<id>) and every page still works.

const rail = document.querySelector(".people");
const panels = [...document.querySelectorAll("[data-person-list]")];
const firstPerson = panels[0]?.dataset.personList;

function selectPerson(id) {
  for (const panel of panels) {
    panel.hidden = panel.dataset.personList !== id;
  }
  if (!rail) return;
  for (const link of rail.querySelectorAll("[data-person]")) {
    const active = link.dataset.person === id;
    link.classList.toggle("active", active);
    link.setAttribute("aria-current", active ? "page" : "false");
  }
}

function personFromUrl() {
  const id = new URLSearchParams(location.search).get("open");
  return panels.some((panel) => panel.dataset.personList === id)
    ? id
    : firstPerson;
}

// All lists are already in the document. Switch locally, keep the URL
// shareable, and let the browser's back/forward buttons work.
rail?.addEventListener("click", (event) => {
  const link = event.target.closest("[data-person]");
  if (!link) return;
  if (
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey ||
    event.button !== 0
  ) {
    return;
  }
  event.preventDefault();
  const id = link.dataset.person;
  selectPerson(id);
  history.pushState({ person: id }, "", `?open=${id}`);
});

window.addEventListener("popstate", () => {
  const id = personFromUrl();
  if (id) selectPerson(id);
});

// Claim dialog.
document.addEventListener("click", (event) => {
  const trigger = event.target.closest("[data-dialog]");
  if (trigger) {
    const dialog = document.getElementById(trigger.dataset.dialog);
    if (dialog && !dialog.open) dialog.showModal();
    return;
  }

  // Clicking the backdrop (not the dialog body) closes the dialog.
  if (event.target instanceof HTMLDialogElement) {
    event.target.close();
  }
});

// Claiming updates the row in place, so the list never reloads or scrolls.
document.addEventListener("submit", async (event) => {
  const form = event.target;
  if (!form.matches?.("form[data-claim]")) return;

  event.preventDefault();
  form.closest("dialog")?.close();

  const id = form.dataset.item;
  try {
    const response = await fetch(form.action, { method: "POST" });
    if (!response.ok) throw new Error(String(response.status));

    const fresh = new DOMParser()
      .parseFromString(await response.text(), "text/html")
      .getElementById(`item-${id}`);
    const current = document.getElementById(`item-${id}`);
    if (!fresh || !current) throw new Error("row not found");

    current.replaceWith(fresh);
  } catch {
    form.submit();
  }
});
