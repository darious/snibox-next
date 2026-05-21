// Snibox Next — tiny vanilla JS. Keyboard shortcuts + toast listener.
// HTMX does the rest.

(function () {
    // Tab in textareas inserts 4 spaces instead of moving focus.
    // Shift+Tab dedents one level (up to 4 leading spaces). Plain editor only —
    // never on inputs or selects.
    const TAB = "    ";
    document.addEventListener("keydown", (e) => {
        if (e.key !== "Tab") return;
        const el = e.target;
        if (!el || el.tagName !== "TEXTAREA") return;
        e.preventDefault();
        const start = el.selectionStart;
        const end = el.selectionEnd;
        const v = el.value;

        // Multi-line selection: indent/dedent per line.
        if (start !== end && v.slice(start, end).includes("\n")) {
            const lineStart = v.lastIndexOf("\n", start - 1) + 1;
            const block = v.slice(lineStart, end);
            let next, delta;
            if (e.shiftKey) {
                next = block.replace(/^ {1,4}/gm, "");
                delta = next.length - block.length;
            } else {
                next = block.replace(/^/gm, TAB);
                delta = next.length - block.length;
            }
            el.value = v.slice(0, lineStart) + next + v.slice(end);
            el.selectionStart = lineStart;
            el.selectionEnd = end + delta;
        } else if (e.shiftKey) {
            const lineStart = v.lastIndexOf("\n", start - 1) + 1;
            const lead = v.slice(lineStart, start).match(/^ {1,4}/);
            if (lead) {
                const cut = lead[0].length;
                el.value = v.slice(0, lineStart) + v.slice(lineStart + cut);
                el.selectionStart = el.selectionEnd = start - cut;
            }
        } else {
            el.value = v.slice(0, start) + TAB + v.slice(end);
            el.selectionStart = el.selectionEnd = start + TAB.length;
        }
        // Fire input so HTMX live-preview triggers.
        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("keyup", { bubbles: true }));
    });

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
            if (document.activeElement && document.activeElement.blur) document.activeElement.blur();
            return;
        }
        if (inEditable(document.activeElement)) return;
        switch (e.key) {
            case "/": e.preventDefault(); focusSearch(); break;
            case "n": e.preventDefault(); window.location.href = "/new?type=snippet"; break;
            case "j": e.preventDefault(); moveActive(1); break;
            case "k": e.preventDefault(); moveActive(-1); break;
            case "Enter": {
                const row = activeRow();
                if (row) row.click();
                break;
            }
            case "e": {
                const ta = document.querySelector(".main-body textarea[name='body']");
                if (ta) { e.preventDefault(); ta.focus(); }
                break;
            }
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

    // Autosave status indicator. Driven by HTMX lifecycle events on textareas
    // marked with data-autosave.
    function setStatus(text) {
        const el = document.getElementById("autosave-status");
        if (el) el.textContent = text;
    }
    document.body.addEventListener("htmx:beforeRequest", (e) => {
        if (e.target && e.target.dataset && e.target.dataset.autosave) setStatus("saving…");
    });
    document.body.addEventListener("htmx:afterRequest", (e) => {
        if (e.target && e.target.dataset && e.target.dataset.autosave) setStatus("saved");
    });

    document.body.addEventListener("click", (e) => {
        const copy = e.target.closest("[data-copy-target]");
        if (!copy) return;
        const src = document.getElementById(copy.dataset.copyTarget);
        if (!src) return;
        navigator.clipboard.writeText(src.textContent).then(() => toast("Copied"));
    });
})();
