/* Editors: Note (markdown split), Snippet (code), Link */

const { useState: useS, useEffect: useE, useRef: useR, useMemo: useM, useCallback: useCB } = React;

/* ---------- Markdown render ---------- */
function renderMarkdown(src) {
  if (!window.marked) return src;
  // Configure marked with highlight.js
  if (!renderMarkdown._configured) {
    marked.setOptions({
      breaks: false,
      gfm: true,
      highlight: (code, lang) => {
        if (window.hljs && lang && hljs.getLanguage(lang)) {
          try { return hljs.highlight(code, { language: lang }).value; } catch (_) {}
        }
        return window.hljs ? hljs.highlightAuto(code).value : code;
      },
    });
    renderMarkdown._configured = true;
  }
  try { return marked.parse(src || ""); } catch (_) { return ""; }
}

/* ---------- Note editor: split markdown + preview ---------- */
function NoteEditor({ item, onChange, mobileShowPreview }) {
  const [body, setBody] = useS(item.body || "");
  useE(() => { setBody(item.body || ""); }, [item.id]);

  const html = useM(() => renderMarkdown(body), [body]);

  const onBodyChange = (e) => {
    setBody(e.target.value);
    onChange({ body: e.target.value });
  };

  return (
    <div className={`md-split ${mobileShowPreview ? "show-preview" : ""}`}>
      <div className="md-pane edit-pane">
        <div className="md-pane-head">edit · markdown</div>
        <textarea
          className="md-edit"
          value={body}
          onChange={onBodyChange}
          spellCheck={false}
          placeholder="# Title\n\nWrite markdown…"
        />
      </div>
      <div className="md-pane preview-pane">
        <div className="md-pane-head">preview</div>
        <div className="md-preview" dangerouslySetInnerHTML={{ __html: html }} />
      </div>
    </div>
  );
}

/* ---------- Snippet editor with highlight + line numbers + copy ---------- */
function SnippetEditor({ item, onChange, lineNumbers, wrap, onCopy }) {
  const [editing, setEditing] = useS(false);
  const [body, setBody] = useS(item.body || "");
  const [language, setLanguage] = useS(item.language || "");
  const codeRef = useR(null);
  const editRef = useR(null);

  useE(() => { setBody(item.body || ""); setLanguage(item.language || ""); setEditing(false); }, [item.id]);

  // Highlight code
  useE(() => {
    if (editing) return;
    if (codeRef.current && window.hljs) {
      const el = codeRef.current;
      el.removeAttribute("data-highlighted");
      el.className = language ? `language-${language}` : "";
      el.textContent = body;
      try { hljs.highlightElement(el); } catch (_) {}
    }
  }, [body, language, editing]);

  // Tab insertion in textarea
  const onKeyDown = (e) => {
    if (e.key === "Tab") {
      e.preventDefault();
      const ta = e.target;
      const s = ta.selectionStart, eend = ta.selectionEnd;
      const next = body.slice(0, s) + "  " + body.slice(eend);
      setBody(next);
      onChange({ body: next });
      requestAnimationFrame(() => { ta.selectionStart = ta.selectionEnd = s + 2; });
    }
  };

  const lines = body.split("\n");
  const lineCount = lines.length || 1;

  return (
    <div className={`snippet-wrap ${wrap ? "wrap" : "no-wrap"}`}>
      <div className="snippet-toolbar">
        <span className="lang">{language || "plain"}</span>
        <span style={{color: "var(--text-mute)"}}>·</span>
        <span>{lineCount} {lineCount === 1 ? "line" : "lines"}</span>
        <span className="spacer"/>
        <button className="btn" onClick={onCopy}><I.Copy size={12}/> Copy</button>
        <button
          className={`btn ${editing ? "active" : ""}`}
          onClick={() => setEditing(!editing)}
        >
          {editing ? <><I.Eye size={12}/> Preview</> : <><I.Edit size={12}/> Edit</>}
        </button>
      </div>

      {editing ? (
        <textarea
          ref={editRef}
          className="code-edit"
          value={body}
          onChange={(e) => { setBody(e.target.value); onChange({ body: e.target.value }); }}
          onKeyDown={onKeyDown}
          spellCheck={false}
          autoFocus
        />
      ) : (
        <div className={`code-view ${lineNumbers ? "has-ln" : ""}`}>
          {lineNumbers && (
            <div className="ln-gutter">
              {Array.from({ length: lineCount }).map((_, i) => (
                <span key={i} className="ln">{i + 1}</span>
              ))}
            </div>
          )}
          <pre><code ref={codeRef} className={language ? `language-${language}` : ""}>{body}</code></pre>
        </div>
      )}
    </div>
  );
}

/* ---------- Link viewer/editor ---------- */
function LinkEditor({ item, onChange }) {
  const [url, setUrl] = useS(item.url || "");
  const [body, setBody] = useS(item.body || "");
  useE(() => { setUrl(item.url || ""); setBody(item.body || ""); }, [item.id]);

  let host = "";
  try { host = url ? new URL(url).host : ""; } catch (_) {}

  return (
    <div className="link-view">
      <div className="link-card">
        <div className="link-url-row">
          <I.Link size={15} style={{color: "var(--text-faint)"}}/>
          <input
            className="link-edit"
            value={url}
            onChange={e => { setUrl(e.target.value); onChange({ url: e.target.value }); }}
            placeholder="https://…"
          />
          {url && (
            <a href={url} target="_blank" rel="noopener noreferrer" className="btn ghost-border" style={{textDecoration: "none"}}>
              <I.External size={13}/> Open
            </a>
          )}
        </div>
        {host && <div className="link-host">{host}</div>}
        <hr/>
        <div style={{fontSize: 11, color: "var(--text-mute)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: 6, fontWeight: 500}}>
          Description
        </div>
        <textarea
          className="desc-edit"
          value={body}
          onChange={e => { setBody(e.target.value); onChange({ body: e.target.value }); }}
          placeholder="What's this link for?"
        />
      </div>
    </div>
  );
}

Object.assign(window, { NoteEditor, SnippetEditor, LinkEditor });
