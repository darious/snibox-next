// Snibox Next — tiny vanilla JS. Keyboard shortcuts + toast listener.
// HTMX does the rest.

(function () {
    function inEditable(el) {
        if (!el) return false;
        const tag = el.tagName;
        return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || el.isContentEditable;
    }

    function focusSearch() {
        const s = document.getElementById("search-input");
        if (s) { s.focus(); s.select(); }
    }

    function clickById(id) {
        const el = document.getElementById(id);
        if (el) el.click();
    }

    function listRows() {
        return Array.from(document.querySelectorAll(".list-row"));
    }

    function activeRow() {
        return document.querySelector('.list-row[data-active="true"]') ||
               document.querySelector('.list-row:focus');
    }

    function moveActive(delta) {
        const rows = listRows();
        if (!rows.length) return;
        const cur = activeRow();
        let i = cur ? rows.indexOf(cur) : -1;
        i = Math.max(0, Math.min(rows.length - 1, i + delta));
        rows.forEach(r => r.removeAttribute("data-active"));
        rows[i].setAttribute("data-active", "true");
        rows[i].scrollIntoView({ block: "nearest" });
        rows[i].focus();
    }

    document.addEventListener("keydown", (e) => {
        if (e.metaKey || e.ctrlKey || e.altKey) return;
        if (e.key === "Escape") {
            const m = document.getElementById("modal-host");
            if (m && m.firstElementChild) { m.innerHTML = ""; return; }
            if (document.activeElement && document.activeElement.blur) document.activeElement.blur();
            return;
        }
        if (inEditable(document.activeElement)) return;
        switch (e.key) {
            case "/": e.preventDefault(); focusSearch(); break;
            case "n": e.preventDefault(); clickById("new-btn"); break;
            case "j": e.preventDefault(); moveActive(1); break;
            case "k": e.preventDefault(); moveActive(-1); break;
            case "Enter": {
                const row = activeRow();
                if (row) row.click();
                break;
            }
            case "e": e.preventDefault(); clickById("snippet-edit-toggle"); break;
            case "p": e.preventDefault(); clickById("pin-btn"); break;
        }
    });

    function toast(msg) {
        const host = document.getElementById("toasts");
        if (!host) return;
        const el = document.createElement("div");
        el.className = "toast";
        el.textContent = msg;
        host.appendChild(el);
        setTimeout(() => el.remove(), 2400);
    }

    document.body.addEventListener("toast", (e) => toast(e.detail.value || e.detail));

    document.body.addEventListener("click", (e) => {
        const copy = e.target.closest("[data-copy-target]");
        if (!copy) return;
        const src = document.getElementById(copy.dataset.copyTarget);
        if (!src) return;
        navigator.clipboard.writeText(src.textContent).then(() => toast("Copied"));
    });
})();
