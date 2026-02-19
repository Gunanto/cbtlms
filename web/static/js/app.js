(function () {
  function cookieValue(name) {
    const match = document.cookie.match(
      new RegExp("(?:^|; )" + name + "=([^;]*)"),
    );
    return match ? decodeURIComponent(match[1]) : "";
  }

  function csrfHeaders(method) {
    const headers = { "Content-Type": "application/json" };
    const m = String(method || "GET").toUpperCase();
    if (m === "POST" || m === "PUT" || m === "PATCH" || m === "DELETE") {
      const csrf = cookieValue("cbtlms_csrf");
      if (csrf) headers["X-CSRF-Token"] = csrf;
    }
    return headers;
  }

  function extractAPIError(json, statusText) {
    if (!json || typeof json !== "object") {
      return statusText || "Request failed";
    }
    if (typeof json.error === "string" && json.error.trim()) {
      return json.error.trim();
    }
    if (
      json.error &&
      typeof json.error === "object" &&
      typeof json.error.message === "string" &&
      json.error.message.trim()
    ) {
      return json.error.message.trim();
    }
    if (typeof json.message === "string" && json.message.trim()) {
      return json.message.trim();
    }
    return statusText || "Request failed";
  }

  async function api(path, method, body) {
    const res = await fetch(path, {
      method,
      credentials: "include",
      headers: csrfHeaders(method),
      body: body ? JSON.stringify(body) : undefined,
    });
    const json = await res.json().catch(function () {
      return {};
    });
    if (!res.ok || !json.ok) {
      const msg = extractAPIError(json, res.statusText || "Request failed");
      throw new Error(msg);
    }
    return json.data;
  }

  async function apiMultipart(path, method, formData) {
    const m = String(method || "POST").toUpperCase();
    const headers = {};
    if (m === "POST" || m === "PUT" || m === "PATCH" || m === "DELETE") {
      const csrf = cookieValue("cbtlms_csrf");
      if (csrf) headers["X-CSRF-Token"] = csrf;
    }
    const res = await fetch(path, {
      method: m,
      credentials: "include",
      headers: headers,
      body: formData,
    });
    const json = await res.json().catch(function () {
      return {};
    });
    if (!res.ok || !json.ok) {
      const msg = extractAPIError(json, res.statusText || "Request failed");
      throw new Error(msg);
    }
    return json.data;
  }

  async function meOrNull() {
    try {
      return await api("/api/v1/auth/me", "GET");
    } catch (_) {
      return null;
    }
  }

  function text(el, value) {
    if (el) el.textContent = value;
  }

  function html(el, value) {
    if (el) el.innerHTML = value;
  }

  function escapeHtml(input) {
    return String(input || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function parseJSONLoose(raw, fallback) {
    if (raw === null || typeof raw === "undefined") return fallback;
    if (typeof raw === "object") return raw;
    if (typeof raw === "string") {
      const s = raw.trim();
      if (!s) return fallback;
      try {
        return JSON.parse(s);
      } catch (_) {
        return fallback;
      }
    }
    return fallback;
  }

  function normalizeCredential(input) {
    return String(input || "")
      .normalize("NFKC")
      .replace(/[\u200B-\u200D\uFEFF]/g, "")
      .replace(/\u00A0/g, " ")
      .trim();
  }

  function beginBusy(container, messageEl, loadingText) {
    if (messageEl && loadingText) text(messageEl, loadingText);
    if (!container) return function () {};

    const controls = [];
    const isForm = !!(container.matches && container.matches("form"));
    if (isForm) {
      // Keep input/select/textarea enabled so FormData values remain readable.
      container.querySelectorAll("button").forEach(function (el) {
        controls.push(el);
      });
    } else {
      if (
        container.matches &&
        container.matches("button, input, select, textarea")
      ) {
        controls.push(container);
      }
      container
        .querySelectorAll("button, input, select, textarea")
        .forEach(function (el) {
          controls.push(el);
        });
    }

    const changed = [];
    controls.forEach(function (el) {
      if (!el.disabled) {
        el.disabled = true;
        changed.push(el);
      }
    });
    if (container.setAttribute) container.setAttribute("aria-busy", "true");

    return function done() {
      changed.forEach(function (el) {
        el.disabled = false;
      });
      if (container.removeAttribute) container.removeAttribute("aria-busy");
    };
  }

  async function initTopbarAuth() {
    const userLabel = document.getElementById("nav-user");
    const logoutBtn = document.getElementById("nav-logout");
    if (!userLabel || !logoutBtn) return;

    const user = await meOrNull();
    if (!user) {
      text(userLabel, "");
      logoutBtn.hidden = true;
      return;
    }

    const label =
      (user.full_name || user.username || "User") +
      " (" +
      String(user.role || "user") +
      ")";
    text(userLabel, label);
    logoutBtn.hidden = false;

    logoutBtn.addEventListener("click", async function () {
      const done = beginBusy(logoutBtn, null, "");
      try {
        await api("/api/v1/auth/logout", "POST");
      } catch (_) {
        // Ignore API error and still force local redirect.
      } finally {
        done();
      }
      window.location.href = "/login";
    });
  }

  async function initLoginPage() {
    const form = document.getElementById("login-form");
    if (!form) return;
    const msg = document.getElementById("login-message");
    const passwordInput = document.getElementById("login-password");
    const togglePasswordBtn = document.getElementById("toggle-login-password");

    if (passwordInput && togglePasswordBtn) {
      togglePasswordBtn.addEventListener("click", function () {
        const show = passwordInput.type === "password";
        passwordInput.type = show ? "text" : "password";
        togglePasswordBtn.setAttribute("aria-pressed", show ? "true" : "false");
        togglePasswordBtn.setAttribute(
          "aria-label",
          show ? "Sembunyikan password" : "Tampilkan password",
        );
        togglePasswordBtn.textContent = show ? "Sembunyikan" : "Lihat";
      });
    }

    const user = await meOrNull();
    if (user) {
      window.location.href = "/simulasi";
      return;
    }

    form.addEventListener("submit", async function (e) {
      e.preventDefault();
      const done = beginBusy(form, msg, "Memproses login...");
      const fd = new FormData(form);
      try {
        const identifier = normalizeCredential(fd.get("identifier"));
        const password = normalizeCredential(fd.get("password"));
        await api("/api/v1/auth/login-password", "POST", {
          identifier: identifier,
          password: password,
        });
        window.location.href = "/simulasi";
      } catch (err) {
        text(msg, "Login gagal: " + err.message);
      } finally {
        done();
      }
    });
  }

  async function initSimulasiPage() {
    const form = document.getElementById("simulasi-form");
    if (!form) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }

    const msg = document.getElementById("simulasi-message");
    const levelSelect = document.getElementById("level-select");
    const typeSelect = document.getElementById("type-select");
    const subjectSelect = document.getElementById("subject-select");
    const examSelect = document.getElementById("exam-select");
    const startBtn = document.getElementById("start-btn");

    let subjects = [];
    const initDone = beginBusy(form, msg, "Memuat daftar mapel...");
    try {
      subjects = await api("/api/v1/subjects", "GET");
    } catch (err) {
      text(msg, "Gagal memuat mapel: " + err.message);
      initDone();
      return;
    }
    initDone();
    const levels = [
      ...new Set(
        subjects.map(function (s) {
          return s.education_level;
        }),
      ),
    ];

    levels.forEach(function (lvl) {
      const o = document.createElement("option");
      o.value = lvl;
      o.textContent = lvl;
      levelSelect.appendChild(o);
    });

    function resetSelect(sel, placeholder) {
      sel.innerHTML = "";
      const o = document.createElement("option");
      o.value = "";
      o.textContent = placeholder;
      sel.appendChild(o);
      sel.disabled = true;
    }

    levelSelect.addEventListener("change", function () {
      resetSelect(typeSelect, "Pilih jenis...");
      resetSelect(subjectSelect, "Pilih mapel...");
      resetSelect(examSelect, "Pilih exam...");
      startBtn.disabled = true;

      const level = levelSelect.value;
      if (!level) return;

      const types = [
        ...new Set(
          subjects
            .filter(function (s) {
              return s.education_level === level;
            })
            .map(function (s) {
              return s.subject_type;
            }),
        ),
      ];

      typeSelect.disabled = false;
      types.forEach(function (t) {
        const o = document.createElement("option");
        o.value = t;
        o.textContent = t;
        typeSelect.appendChild(o);
      });
    });

    typeSelect.addEventListener("change", function () {
      resetSelect(subjectSelect, "Pilih mapel...");
      resetSelect(examSelect, "Pilih exam...");
      startBtn.disabled = true;

      const level = levelSelect.value;
      const type = typeSelect.value;
      if (!level || !type) return;

      subjectSelect.disabled = false;
      subjects
        .filter(function (s) {
          return s.education_level === level && s.subject_type === type;
        })
        .forEach(function (s) {
          const o = document.createElement("option");
          o.value = String(s.id);
          o.textContent = s.name;
          subjectSelect.appendChild(o);
        });
    });

    subjectSelect.addEventListener("change", async function () {
      resetSelect(examSelect, "Pilih exam...");
      startBtn.disabled = true;

      const subjectID = subjectSelect.value;
      if (!subjectID) return;

      const done = beginBusy(examSelect, msg, "Memuat daftar exam...");
      try {
        const exams = await api(
          "/api/v1/exams?subject_id=" + encodeURIComponent(subjectID),
          "GET",
        );
        examSelect.disabled = false;
        exams.forEach(function (x) {
          const o = document.createElement("option");
          o.value = String(x.id);
          o.textContent = x.code + " - " + x.title;
          examSelect.appendChild(o);
        });
      } catch (err) {
        text(msg, "Gagal load exam: " + err.message);
      } finally {
        done();
      }
    });

    examSelect.addEventListener("change", function () {
      startBtn.disabled = !examSelect.value;
    });

    form.addEventListener("submit", async function (e) {
      e.preventDefault();
      const examID = Number(examSelect.value || 0);
      if (!examID) return;

      const done = beginBusy(form, msg, "Menyiapkan attempt...");
      try {
        const payload = { exam_id: examID };
        if (user.role === "admin" || user.role === "proktor") {
          payload.student_id = user.id;
        }
        const out = await api("/api/v1/attempts/start", "POST", payload);
        window.location.href = "/ujian/" + out.id;
      } catch (err) {
        text(msg, "Gagal start attempt: " + err.message);
      } finally {
        done();
      }
    });
  }

  function readSelectedFromPayload(payload) {
    if (!payload || typeof payload !== "object") return [];
    const v = payload.selected;
    if (typeof v === "string") return v ? [v] : [];
    if (Array.isArray(v))
      return v.filter(function (x) {
        return typeof x === "string" && x.trim() !== "";
      });
    return [];
  }

  async function initAttemptPage() {
    const root = document.getElementById("attempt-root");
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }

    const attemptID = Number(root.getAttribute("data-attempt-id") || 0);
    let currentNo = 1;
    let totalQuestions = 1;
    let currentQuestion = null;
    let remainInterval = null;
    let attemptEditable = true;

    const remainingLabel = document.getElementById("remaining-label");
    const message = document.getElementById("attempt-message");
    const questionTitle = document.getElementById("question-title");
    const stimulusBox = document.getElementById("stimulus-box");
    const stemBox = document.getElementById("stem-box");
    const optionsBox = document.getElementById("options-box");
    const doubtCheckbox = document.getElementById("doubt-checkbox");
    const prevBtn = document.getElementById("prev-btn");
    const nextBtn = document.getElementById("next-btn");
    const submitBtn = document.getElementById("submit-btn");
    const actionsBar = document.querySelector(".actions");
    const infoSoalBtn = document.getElementById("info-soal-btn");
    const daftarSoalBtn = document.getElementById("daftar-soal-btn");
    const daftarSoalPanel = document.getElementById("daftar-soal-panel");
    const daftarSoalBody = document.getElementById("daftar-soal-body");
    const eventMessage = document.getElementById("attempt-event-message");
    const questionState = {};
    let eventMessageTimer = null;
    const eventLastAt = {};
    const eventMinIntervalMs = {
      tab_blur: 5000,
      reconnect: 5000,
      fullscreen_exit: 5000,
      rapid_refresh: 15000,
    };
    const pendingEvents = [];
    let flushingEvents = false;

    function formatRemain(secs) {
      const s = Math.max(0, Number(secs || 0));
      const h = Math.floor(s / 3600);
      const m = Math.floor((s % 3600) / 60);
      const ss = s % 60;
      return (
        String(h).padStart(2, "0") +
        ":" +
        String(m).padStart(2, "0") +
        ":" +
        String(ss).padStart(2, "0")
      );
    }

    function isAnsweredPayload(qType, payload) {
      if (qType === "benar_salah_pernyataan") {
        return Array.isArray(payload.answers) && payload.answers.length > 0;
      }
      if (qType === "pg_tunggal") {
        return (
          typeof payload.selected === "string" && payload.selected.trim() !== ""
        );
      }
      if (qType === "multi_jawaban") {
        return Array.isArray(payload.selected) && payload.selected.length > 0;
      }
      return false;
    }

    function isAnswered(questionLike) {
      if (!questionLike) return false;
      const qType = String(questionLike.question_type || "").toLowerCase();
      const payload = parseJSONLoose(questionLike.answer_payload, {});
      return isAnsweredPayload(qType, payload);
    }

    function ensureQuestionList() {
      if (!daftarSoalBody) return;
      if (daftarSoalBody.childElementCount === totalQuestions) return;
      daftarSoalBody.innerHTML = "";
      for (let i = 1; i <= totalQuestions; i += 1) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "qno-btn";
        btn.setAttribute("data-qno", String(i));
        btn.textContent = String(i);
        daftarSoalBody.appendChild(btn);
      }
    }

    function renderQuestionListState() {
      if (!daftarSoalBody) return;
      daftarSoalBody.querySelectorAll(".qno-btn").forEach(function (btn) {
        const no = Number(btn.getAttribute("data-qno") || 0);
        const state = questionState[no] || {};
        btn.classList.toggle("current", no === currentNo);
        btn.classList.toggle("answered", !!state.answered);
        btn.classList.toggle("doubt", !!state.doubt);
      });
    }

    function closeQuestionList() {
      if (!daftarSoalPanel || !daftarSoalBtn) return;
      daftarSoalPanel.hidden = true;
      daftarSoalBtn.setAttribute("aria-expanded", "false");
    }

    function openQuestionList() {
      if (!daftarSoalPanel || !daftarSoalBtn) return;
      ensureQuestionList();
      renderQuestionListState();
      daftarSoalPanel.hidden = false;
      daftarSoalBtn.setAttribute("aria-expanded", "true");
    }

    async function loadSummary() {
      const sum = await api("/api/v1/attempts/" + attemptID, "GET");
      totalQuestions = Number(sum.total_questions || 1);
      attemptEditable = String(sum.status || "") === "in_progress";
      ensureQuestionList();
      let remain = Number(sum.remaining_secs || 0);
      text(remainingLabel, formatRemain(remain));
      if (remainInterval) clearInterval(remainInterval);
      remainInterval = setInterval(function () {
        remain = Math.max(0, remain - 1);
        text(remainingLabel, formatRemain(remain));
      }, 1000);
      return sum;
    }

    function applyAttemptReadonlyState(summaryStatus) {
      attemptEditable = false;
      prevBtn.disabled = true;
      nextBtn.disabled = true;
      submitBtn.disabled = true;
      submitBtn.hidden = true;
      doubtCheckbox.disabled = true;
      optionsBox
        .querySelectorAll("input, button, select, textarea")
        .forEach(function (el) {
          el.disabled = true;
        });
      html(
        message,
        "Attempt sudah " +
          escapeHtml(String(summaryStatus || "final")) +
          ' dan tidak bisa diedit. Buka <a href="/hasil/' +
          encodeURIComponent(String(attemptID)) +
          '">halaman hasil</a>.',
      );
    }

    function syncSubmitVisibility() {
      if (!attemptEditable) {
        submitBtn.hidden = true;
        if (actionsBar) actionsBar.classList.remove("with-submit");
        return;
      }
      const isLast = currentNo === totalQuestions;
      submitBtn.hidden = !isLast;
      if (actionsBar) actionsBar.classList.toggle("with-submit", isLast);
    }

    function notifyEvent(msg) {
      text(eventMessage, msg);
      if (eventMessageTimer) {
        clearTimeout(eventMessageTimer);
      }
      eventMessageTimer = setTimeout(function () {
        text(eventMessage, "");
      }, 2500);
    }

    function canSendEvent(eventType) {
      const now = Date.now();
      const minInterval = eventMinIntervalMs[eventType] || 5000;
      const last = eventLastAt[eventType] || 0;
      if (now - last < minInterval) {
        return false;
      }
      eventLastAt[eventType] = now;
      return true;
    }

    async function flushAttemptEvents() {
      if (flushingEvents) return;
      flushingEvents = true;
      try {
        while (pendingEvents.length > 0) {
          const item = pendingEvents[0];
          await api("/api/v1/attempts/" + attemptID + "/events", "POST", item);
          pendingEvents.shift();
        }
      } catch (_) {
        // Non-blocking: anti-cheat logging should never break exam flow.
      } finally {
        flushingEvents = false;
      }
    }

    function queueAttemptEvent(eventType, payload, userNotice) {
      if (!canSendEvent(eventType)) return;
      pendingEvents.push({
        event_type: eventType,
        payload: payload || {},
        client_ts: new Date().toISOString(),
      });
      if (userNotice) notifyEvent(userNotice);
      flushAttemptEvents();
    }

    function renderOptions(q) {
      const payload = parseJSONLoose(q.answer_payload, {});

      const selected = new Set(readSelectedFromPayload(payload));
      const qType = String(q.question_type || "").toLowerCase();

      if (qType === "benar_salah_pernyataan") {
        const key = parseJSONLoose(q.answer_key, {});
        const statements = Array.isArray(key.statements) ? key.statements : [];
        if (!statements.length) {
          html(optionsBox, '<p class="muted">Pernyataan tidak tersedia.</p>');
          return;
        }

        const existing = {};
        if (Array.isArray(payload.answers)) {
          payload.answers.forEach(function (a) {
            if (a && typeof a.id === "string" && typeof a.value === "boolean") {
              existing[a.id] = a.value;
            }
          });
        }

        const rows = statements.map(function (s, idx) {
          const id = String((s && s.id) || "S" + (idx + 1));
          const label =
            (s && (s.text || s.statement || s.label || s.statement_html)) || id;
          const yesChecked = existing[id] === true ? "checked" : "";
          const noChecked = existing[id] === false ? "checked" : "";
          return (
            "<tr>" +
            '<td class="bs-statement">' +
            escapeHtml(String(label)) +
            "</td>" +
            '<td class="bs-choice">' +
            '<input type="radio" name="bs_' +
            escapeHtml(id) +
            '" value="true" ' +
            yesChecked +
            ">" +
            "</td>" +
            '<td class="bs-choice">' +
            '<input type="radio" name="bs_' +
            escapeHtml(id) +
            '" value="false" ' +
            noChecked +
            ">" +
            "</td>" +
            "</tr>"
          );
        });
        html(
          optionsBox,
          '<table class="bs-table"><thead><tr><th>Pernyataan</th><th>Benar</th><th>Salah</th></tr></thead><tbody>' +
            rows.join("") +
            "</tbody></table>",
        );
        return;
      }

      const multiple = qType === "multi_jawaban";
      const inputType = multiple ? "checkbox" : "radio";
      const rows = (q.options || []).map(function (opt) {
        const checked = selected.has(opt.option_key) ? "checked" : "";
        return [
          '<label class="option-row">',
          '<input type="' +
            inputType +
            '" name="option" value="' +
            escapeHtml(opt.option_key) +
            '" ' +
            checked +
            ">",
          "<div>" + opt.option_html + "</div>",
          "</label>",
        ].join("");
      });
      html(optionsBox, rows.join(""));
    }

    function collectPayload(q) {
      const qType = String(q.question_type || "").toLowerCase();
      if (qType === "benar_salah_pernyataan") {
        const key = parseJSONLoose(q.answer_key, {});
        const statements = Array.isArray(key.statements) ? key.statements : [];
        const answers = [];

        statements.forEach(function (s, idx) {
          const id = String((s && s.id) || "S" + (idx + 1));
          const selected = document.querySelector(
            'input[name="bs_' + CSS.escape(id) + '"]:checked',
          );
          if (
            selected &&
            (selected.value === "true" || selected.value === "false")
          ) {
            answers.push({ id: id, value: selected.value === "true" });
          }
        });
        return { answers: answers };
      }

      const selectedNodes = optionsBox.querySelectorAll(
        'input[name="option"]:checked',
      );
      const selected = Array.from(selectedNodes)
        .map(function (n) {
          return String(n.value || "").trim();
        })
        .filter(Boolean);
      if (qType === "pg_tunggal") {
        return { selected: selected[0] || "" };
      }
      return { selected: selected };
    }

    async function saveCurrent() {
      if (!attemptEditable) return;
      if (!currentQuestion) return;
      const payload = collectPayload(currentQuestion);
      await api(
        "/api/v1/attempts/" +
          attemptID +
          "/answers/" +
          currentQuestion.question_id,
        "PUT",
        {
          answer_payload: payload,
          is_doubt: !!doubtCheckbox.checked,
        },
      );
      questionState[currentNo] = {
        answered: isAnsweredPayload(
          String(currentQuestion.question_type || "").toLowerCase(),
          payload,
        ),
        doubt: !!doubtCheckbox.checked,
      };
      renderQuestionListState();
    }

    async function loadQuestion(no) {
      const q = await api(
        "/api/v1/attempts/" + attemptID + "/questions/" + no,
        "GET",
      );
      currentQuestion = q;
      currentNo = no;
      questionState[no] = {
        answered: isAnswered(q),
        doubt: !!q.is_doubt,
      };
      syncSubmitVisibility();
      renderQuestionListState();
      text(questionTitle, "Soal nomor " + no + " dari " + totalQuestions);
      html(stemBox, q.stem_html || '<p class="muted">Soal tidak tersedia.</p>');
      html(
        stimulusBox,
        q.stimulus_html || '<p class="muted">Tidak ada stimulus.</p>',
      );
      doubtCheckbox.checked = !!q.is_doubt;
      renderOptions(q);
      if (!attemptEditable) {
        applyAttemptReadonlyState("final");
        return;
      }
      text(message, "");
    }

    if (infoSoalBtn) {
      infoSoalBtn.addEventListener("click", function () {
        text(
          message,
          "Gunakan Daftar Soal untuk lompat nomor. Soal saat ini: " +
            currentNo +
            "/" +
            totalQuestions +
            ".",
        );
      });
    }

    if (daftarSoalBtn) {
      daftarSoalBtn.addEventListener("click", function () {
        if (daftarSoalPanel && daftarSoalPanel.hidden) {
          openQuestionList();
          return;
        }
        closeQuestionList();
      });
    }

    if (daftarSoalBody) {
      daftarSoalBody.addEventListener("click", async function (e) {
        const btn = e.target.closest(".qno-btn");
        if (!btn) return;
        const targetNo = Number(btn.getAttribute("data-qno") || 0);
        if (!targetNo || targetNo === currentNo) {
          closeQuestionList();
          return;
        }
        const done = beginBusy(
          btn,
          message,
          "Memuat soal nomor " + targetNo + "...",
        );
        try {
          if (attemptEditable) {
            await saveCurrent();
          }
          await loadQuestion(targetNo);
          closeQuestionList();
        } catch (err) {
          if (String(err.message || "").includes("attempt is not editable")) {
            applyAttemptReadonlyState("submitted");
            return;
          }
          text(message, "Gagal pindah soal: " + err.message);
        } finally {
          done();
        }
      });
    }

    document.addEventListener("click", function (e) {
      if (!daftarSoalPanel || daftarSoalPanel.hidden) return;
      if (
        (daftarSoalBtn && daftarSoalBtn.contains(e.target)) ||
        daftarSoalPanel.contains(e.target)
      ) {
        return;
      }
      closeQuestionList();
    });

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") {
        closeQuestionList();
      }
    });

    document
      .getElementById("prev-btn")
      .addEventListener("click", async function () {
        if (!attemptEditable) return;
        if (currentNo <= 1) return;
        const done = beginBusy(this, message, "Memuat soal sebelumnya...");
        try {
          await saveCurrent();
          await loadQuestion(currentNo - 1);
        } catch (err) {
          if (String(err.message || "").includes("attempt is not editable")) {
            applyAttemptReadonlyState("submitted");
            return;
          }
          text(message, "Gagal pindah soal: " + err.message);
        } finally {
          done();
        }
      });

    document
      .getElementById("next-btn")
      .addEventListener("click", async function () {
        if (!attemptEditable) return;
        if (currentNo >= totalQuestions) return;
        const done = beginBusy(this, message, "Memuat soal berikutnya...");
        try {
          await saveCurrent();
          await loadQuestion(currentNo + 1);
        } catch (err) {
          if (String(err.message || "").includes("attempt is not editable")) {
            applyAttemptReadonlyState("submitted");
            return;
          }
          text(message, "Gagal pindah soal: " + err.message);
        } finally {
          done();
        }
      });

    document
      .getElementById("submit-btn")
      .addEventListener("click", async function () {
        if (!attemptEditable) return;
        if (!window.confirm("Yakin submit final?")) return;
        const done = beginBusy(this, message, "Memproses submit final...");
        try {
          await saveCurrent();
          await api("/api/v1/attempts/" + attemptID + "/submit", "POST");
          window.location.href = "/hasil/" + attemptID;
        } catch (err) {
          if (String(err.message || "").includes("attempt is not editable")) {
            applyAttemptReadonlyState("submitted");
            return;
          }
          text(message, "Gagal submit: " + err.message);
        } finally {
          done();
        }
      });

    optionsBox.addEventListener("change", async function () {
      if (!attemptEditable) return;
      text(message, "Menyimpan jawaban...");
      try {
        await saveCurrent();
        text(message, "Tersimpan otomatis.");
      } catch (err) {
        if (String(err.message || "").includes("attempt is not editable")) {
          applyAttemptReadonlyState("submitted");
          return;
        }
        text(message, "Autosave gagal: " + err.message);
      }
    });

    doubtCheckbox.addEventListener("change", async function () {
      if (!attemptEditable) return;
      text(message, "Menyimpan status ragu-ragu...");
      try {
        await saveCurrent();
        text(message, "Status ragu-ragu tersimpan.");
      } catch (err) {
        if (String(err.message || "").includes("attempt is not editable")) {
          applyAttemptReadonlyState("submitted");
          return;
        }
        text(message, "Simpan ragu-ragu gagal: " + err.message);
      }
    });

    document.addEventListener("visibilitychange", function () {
      if (document.visibilityState === "hidden") {
        queueAttemptEvent(
          "tab_blur",
          {
            reason: "visibility_hidden",
            visibility_state: document.visibilityState,
          },
          "Aktivitas tab keluar terdeteksi.",
        );
        return;
      }
      queueAttemptEvent("reconnect", {
        reason: "visibility_visible",
        visibility_state: document.visibilityState,
      });
    });

    window.addEventListener("blur", function () {
      queueAttemptEvent(
        "tab_blur",
        {
          reason: "window_blur",
          visibility_state: document.visibilityState,
        },
        "Perpindahan fokus terdeteksi.",
      );
    });

    window.addEventListener("focus", function () {
      queueAttemptEvent("reconnect", {
        reason: "window_focus",
        visibility_state: document.visibilityState,
      });
    });

    window.addEventListener("offline", function () {
      notifyEvent("Koneksi terputus, sistem akan sinkron saat tersambung.");
    });

    window.addEventListener("online", function () {
      queueAttemptEvent(
        "reconnect",
        {
          reason: "online",
          visibility_state: document.visibilityState,
        },
        "Koneksi kembali normal.",
      );
    });

    document.addEventListener("fullscreenchange", function () {
      if (document.fullscreenElement) return;
      queueAttemptEvent(
        "fullscreen_exit",
        {
          reason: "fullscreen_exit",
          visibility_state: document.visibilityState,
        },
        "Keluar dari mode fullscreen terdeteksi.",
      );
    });

    try {
      text(message, "Memuat attempt...");
      const summary = await loadSummary();
      syncSubmitVisibility();
      await loadQuestion(1);
      if (!attemptEditable) {
        applyAttemptReadonlyState(summary.status);
      }
    } catch (err) {
      text(message, "Gagal memuat attempt: " + err.message);
    }
  }

  async function initResultPage() {
    const root = document.getElementById("result-root");
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }

    const attemptID = Number(root.getAttribute("data-attempt-id") || 0);
    const summaryBox = document.getElementById("result-summary");
    const tbody = document.querySelector("#result-table tbody");

    try {
      const out = await api("/api/v1/attempts/" + attemptID + "/result", "GET");
      const s = out.summary;
      html(
        summaryBox,
        "<strong>Status:</strong> " +
          escapeHtml(s.status) +
          " | <strong>Skor:</strong> " +
          escapeHtml(String(s.score)) +
          " | <strong>Benar:</strong> " +
          escapeHtml(String(s.total_correct)) +
          " | <strong>Salah:</strong> " +
          escapeHtml(String(s.total_wrong)) +
          " | <strong>Kosong:</strong> " +
          escapeHtml(String(s.total_unanswered)),
      );

      tbody.innerHTML = "";
      out.items.forEach(function (it, idx) {
        const tr = document.createElement("tr");
        const breakdown =
          Array.isArray(it.breakdown) && it.breakdown.length
            ? "<br><small>" +
              it.breakdown
                .map(function (b) {
                  return escapeHtml(b.id + ":" + (b.correct ? "ok" : "x"));
                })
                .join(", ") +
              "</small>"
            : "";
        tr.innerHTML = [
          "<td>" + (idx + 1) + "</td>",
          "<td>" + escapeHtml(String(it.question_id)) + "</td>",
          "<td>" + escapeHtml((it.selected || []).join(", ")) + "</td>",
          "<td>" + escapeHtml((it.correct || []).join(", ")) + "</td>",
          "<td>" + escapeHtml(it.reason || "") + breakdown + "</td>",
          "<td>" + escapeHtml(String(it.earned_score || 0)) + "</td>",
        ].join("");
        tbody.appendChild(tr);
      });
    } catch (err) {
      text(summaryBox, "Gagal memuat hasil: " + err.message);
    }
  }

  async function initAdminPage() {
    const root = document.querySelector('[data-page="admin_content"]');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }
    if (!["admin", "proktor"].includes(String(user.role || ""))) {
      alert("Halaman admin hanya untuk admin/proktor");
      window.location.href = "/";
      return;
    }

    const message = document.getElementById("admin-message");
    const regSummary = document.getElementById("admin-registration-summary");
    const regBody = document.getElementById("admin-registration-body");
    const regOut = document.getElementById("admin-registration-output");
    const masterOut = document.getElementById("admin-master-output");
    const prevPageBtn = document.getElementById("admin-prev-page-btn");
    const nextPageBtn = document.getElementById("admin-next-page-btn");
    const pageInfo = document.getElementById("admin-page-info");
    const regFilterForm = document.getElementById(
      "admin-registration-filter-form",
    );
    const statAdmin = document.getElementById("admin-stat-admin");
    const statProktor = document.getElementById("admin-stat-proktor");
    const statGuru = document.getElementById("admin-stat-guru");
    const statSiswa = document.getElementById("admin-stat-siswa");
    const statSekolah = document.getElementById("admin-stat-sekolah");
    const exportUsersBtn = document.getElementById("admin-user-export-btn");
    const importUsersBtn = document.getElementById("admin-user-import-btn");
    const addUserBtn = document.getElementById("admin-user-add-btn");
    const userDetailDialog = document.getElementById(
      "admin-user-detail-dialog",
    );
    const userDetailOut = document.getElementById("admin-user-detail-output");
    const userCreateDialog = document.getElementById(
      "admin-user-create-dialog",
    );
    const userCreateForm = document.getElementById("admin-user-create-form");
    const userCreateCancelBtn = document.getElementById(
      "admin-user-create-cancel-btn",
    );
    const userUpdateDialog = document.getElementById(
      "admin-user-update-dialog",
    );
    const userUpdateForm = document.getElementById("admin-user-update-form");
    const userUpdateCancelBtn = document.getElementById(
      "admin-user-update-cancel-btn",
    );
    const userDeleteDialog = document.getElementById(
      "admin-user-delete-dialog",
    );
    const userDeleteForm = document.getElementById("admin-user-delete-form");
    const userDeleteLabel = document.getElementById("admin-user-delete-label");
    const userDeleteCancelBtn = document.getElementById(
      "admin-user-delete-cancel-btn",
    );
    const userImportDialog = document.getElementById(
      "admin-user-import-dialog",
    );
    const userImportForm = document.getElementById("admin-user-import-form");
    const userImportOut = document.getElementById("admin-user-import-output");
    const userImportCancelBtn = document.getElementById(
      "admin-user-import-cancel-btn",
    );
    const userImportErrorDialog = document.getElementById(
      "admin-user-import-error-dialog",
    );
    const userImportErrorOut = document.getElementById(
      "admin-user-import-error-output",
    );
    const userImportErrorCloseBtn = document.getElementById(
      "admin-user-import-error-close-btn",
    );
    const masterMessage = document.getElementById("admin-master-message");
    const masterRefreshBtn = document.getElementById(
      "admin-master-refresh-btn",
    );
    const levelAddBtn = document.getElementById("admin-level-add-btn");
    const schoolAddBtn = document.getElementById("admin-school-add-btn");
    const classAddBtn = document.getElementById("admin-class-add-btn");
    const levelTableBody = document.getElementById("admin-level-table-body");
    const schoolTableBody = document.getElementById("admin-school-table-body");
    const classTableBody = document.getElementById("admin-class-table-body");
    const levelDialog = document.getElementById("admin-level-dialog");
    const levelDialogForm = document.getElementById("admin-level-dialog-form");
    const levelDialogTitle = document.getElementById(
      "admin-level-dialog-title",
    );
    const levelDialogCancelBtn = document.getElementById(
      "admin-level-dialog-cancel-btn",
    );
    const schoolDialog = document.getElementById("admin-school-dialog");
    const schoolDialogForm = document.getElementById(
      "admin-school-dialog-form",
    );
    const schoolDialogTitle = document.getElementById(
      "admin-school-dialog-title",
    );
    const schoolDialogCancelBtn = document.getElementById(
      "admin-school-dialog-cancel-btn",
    );
    const classDialog = document.getElementById("admin-class-dialog");
    const classDialogForm = document.getElementById("admin-class-dialog-form");
    const classDialogTitle = document.getElementById(
      "admin-class-dialog-title",
    );
    const classDialogCancelBtn = document.getElementById(
      "admin-class-dialog-cancel-btn",
    );
    const masterDeleteDialog = document.getElementById(
      "admin-master-delete-dialog",
    );
    const masterDeleteForm = document.getElementById(
      "admin-master-delete-form",
    );
    const masterDeleteLabel = document.getElementById(
      "admin-master-delete-label",
    );
    const masterDeleteCancelBtn = document.getElementById(
      "admin-master-delete-cancel-btn",
    );
    const menuButtons = root.querySelectorAll("[data-admin-menu]");
    const usersPanel = document.getElementById("admin-users-panel");
    const masterPanel = document.getElementById("admin-master-panel");
    let currentRole = "";
    let currentQuery = "";
    let currentLimit = 50;
    let currentOffset = 0;
    let lastLoadedCount = 0;
    const usersByID = {};
    let schoolsCache = [];
    const classCacheBySchool = {};
    const masterSummaryState = {
      levelsByID: {},
      schoolsByID: {},
      classesByID: {},
    };
    let filterDebounceTimer = null;

    function setMsg(msg) {
      text(message, msg);
    }

    function pretty(el, data) {
      if (el) el.textContent = JSON.stringify(data, null, 2);
    }

    function fmtDate(raw) {
      if (!raw) return "-";
      const d = new Date(raw);
      if (Number.isNaN(d.getTime())) return String(raw);
      return d.toLocaleString("id-ID");
    }

    function isAdmin() {
      return String(user.role || "") === "admin";
    }

    function toNullableNumber(input) {
      const n = Number(input || 0);
      if (!Number.isFinite(n) || n <= 0) return null;
      return n;
    }

    function schoolLabel(it) {
      const name = String((it && it.name) || "").trim();
      const code = String((it && it.code) || "").trim();
      if (!name) return "-";
      return code ? name + " (" + code + ")" : name;
    }

    function classLabel(it) {
      const name = String((it && it.name) || "").trim();
      const grade = String((it && it.grade_level) || "").trim();
      if (!name) return "-";
      return grade ? grade + " - " + name : name;
    }

    function resetClassSelect(selectEl, placeholder) {
      if (!selectEl) return;
      selectEl.innerHTML = "";
      const opt = document.createElement("option");
      opt.value = "";
      opt.textContent = placeholder || "Pilih Sekolah Dulu";
      selectEl.appendChild(opt);
      selectEl.disabled = true;
    }

    function fillSchoolSelect(selectEl, selectedID) {
      if (!selectEl) return;
      const selected = Number(selectedID || 0);
      selectEl.innerHTML = "";
      const emptyOpt = document.createElement("option");
      emptyOpt.value = "";
      emptyOpt.textContent = "Tanpa Sekolah";
      selectEl.appendChild(emptyOpt);
      schoolsCache.forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id);
        opt.textContent = schoolLabel(it);
        if (selected > 0 && Number(it.id) === selected) opt.selected = true;
        selectEl.appendChild(opt);
      });
    }

    function fillSchoolSelectRequired(selectEl, selectedID) {
      if (!selectEl) return;
      const selected = Number(selectedID || 0);
      selectEl.innerHTML = "";
      const emptyOpt = document.createElement("option");
      emptyOpt.value = "";
      emptyOpt.textContent = "Pilih Sekolah";
      selectEl.appendChild(emptyOpt);
      schoolsCache.forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id);
        opt.textContent = schoolLabel(it);
        if (selected > 0 && Number(it.id) === selected) opt.selected = true;
        selectEl.appendChild(opt);
      });
    }

    function fillClassSelect(selectEl, classes, selectedID) {
      if (!selectEl) return;
      const selected = Number(selectedID || 0);
      selectEl.innerHTML = "";
      const emptyOpt = document.createElement("option");
      emptyOpt.value = "";
      emptyOpt.textContent = "Tanpa Kelas";
      selectEl.appendChild(emptyOpt);
      (classes || []).forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id);
        opt.textContent = classLabel(it);
        if (selected > 0 && Number(it.id) === selected) opt.selected = true;
        selectEl.appendChild(opt);
      });
      selectEl.disabled = false;
    }

    function pickFirstClassIfAny(selectEl) {
      if (!selectEl || selectEl.disabled) return;
      if (selectEl.value) return;
      const firstClassOption = Array.from(selectEl.options || []).find(
        function (opt) {
          return String(opt.value || "").trim() !== "";
        },
      );
      if (firstClassOption) {
        selectEl.value = String(firstClassOption.value);
      }
    }

    async function loadSchoolsForUserForms() {
      const out = await api("/api/v1/admin/schools?all=1", "GET");
      schoolsCache = Array.isArray(out) ? out : [];
    }

    async function loadClassesBySchool(schoolID) {
      const sid = Number(schoolID || 0);
      if (sid <= 0) return [];
      if (Array.isArray(classCacheBySchool[String(sid)])) {
        return classCacheBySchool[String(sid)];
      }
      const out = await api(
        "/api/v1/admin/classes?all=1&school_id=" +
          encodeURIComponent(String(sid)),
        "GET",
      );
      const items = Array.isArray(out) ? out : [];
      classCacheBySchool[String(sid)] = items;
      return items;
    }

    function parseDownloadFilename(disposition, fallback) {
      const src = String(disposition || "");
      const utfMatch = src.match(/filename\*=UTF-8''([^;]+)/i);
      if (utfMatch && utfMatch[1]) {
        try {
          return decodeURIComponent(utfMatch[1]);
        } catch (_) {
          return utfMatch[1];
        }
      }
      const basicMatch = src.match(/filename="?([^";]+)"?/i);
      if (basicMatch && basicMatch[1]) return basicMatch[1];
      return fallback;
    }

    async function exportUsersExcelFile() {
      const params = new URLSearchParams();
      if (currentRole) params.set("role", currentRole);
      if (currentQuery) params.set("q", currentQuery);
      const qs = params.toString() ? "?" + params.toString() : "";
      const res = await fetch("/api/v1/admin/users/export" + qs, {
        method: "GET",
        credentials: "include",
      });
      if (!res.ok) {
        const errJSON = await res.json().catch(function () {
          return {};
        });
        const msg = extractAPIError(
          errJSON,
          res.statusText || "Request failed",
        );
        throw new Error(msg);
      }

      const blob = await res.blob();
      const filename = parseDownloadFilename(
        res.headers.get("Content-Disposition"),
        "daftar_pengguna.xlsx",
      );
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    }

    function activateMenu(target) {
      if (usersPanel) usersPanel.hidden = target !== "users";
      if (masterPanel) masterPanel.hidden = target !== "master";
      menuButtons.forEach(function (btn) {
        const isActive = btn.getAttribute("data-admin-menu") === target;
        btn.classList.toggle("active", isActive);
      });
    }

    async function loadEducationLevels(showAll) {
      const qs = showAll ? "?all=1" : "";
      const out = await api("/api/v1/levels" + qs, "GET");
      return Array.isArray(out) ? out : [];
    }

    function renderMasterSummaryTable(levels, schools, classes) {
      const canEditMaster = isAdmin();
      const levelItems = Array.isArray(levels) ? levels : [];
      const schoolItems = Array.isArray(schools) ? schools : [];
      const classItems = Array.isArray(classes) ? classes : [];
      masterSummaryState.levelsByID = {};
      masterSummaryState.schoolsByID = {};
      masterSummaryState.classesByID = {};
      levelItems.forEach(function (it) {
        masterSummaryState.levelsByID[String(it.id || "")] = it;
      });
      schoolItems.forEach(function (it) {
        masterSummaryState.schoolsByID[String(it.id || "")] = it;
      });
      classItems.forEach(function (it) {
        masterSummaryState.classesByID[String(it.id || "")] = it;
      });
      if (levelTableBody) {
        if (!levelItems.length) {
          levelTableBody.innerHTML =
            '<tr><td colspan="4" class="muted">Belum ada data jenjang.</td></tr>';
        } else {
          levelTableBody.innerHTML = "";
          levelItems.forEach(function (it) {
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
              "<td>" + (it.is_active ? "aktif" : "nonaktif") + "</td>",
              "<td>" +
                (canEditMaster
                  ? '<span class="action-icons">' +
                    '<button class="icon-only-btn" type="button" data-master-entity="level" data-master-action="edit" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Ubah jenjang" aria-label="Ubah jenjang"><svg viewBox="0 0 24 24" focusable="false"><path d="M4 20h4l10-10-4-4L4 16v4z"/><path d="M12 6l4 4"/></svg></button>' +
                    '<button class="icon-only-btn danger" type="button" data-master-entity="level" data-master-action="delete" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Hapus jenjang" aria-label="Hapus jenjang"><svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg></button>' +
                    "</span>"
                  : "-") +
                "</td>",
            ].join("");
            levelTableBody.appendChild(tr);
          });
        }
      }
      if (schoolTableBody) {
        if (!schoolItems.length) {
          schoolTableBody.innerHTML =
            '<tr><td colspan="5" class="muted">Belum ada data sekolah.</td></tr>';
        } else {
          schoolTableBody.innerHTML = "";
          schoolItems.forEach(function (it) {
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
              "<td>" + escapeHtml(String(it.code || "-")) + "</td>",
              "<td>" + escapeHtml(String(it.address || "-")) + "</td>",
              "<td>" +
                (canEditMaster
                  ? '<span class="action-icons">' +
                    '<button class="icon-only-btn" type="button" data-master-entity="school" data-master-action="edit" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Ubah sekolah" aria-label="Ubah sekolah"><svg viewBox="0 0 24 24" focusable="false"><path d="M4 20h4l10-10-4-4L4 16v4z"/><path d="M12 6l4 4"/></svg></button>' +
                    '<button class="icon-only-btn danger" type="button" data-master-entity="school" data-master-action="delete" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Hapus sekolah" aria-label="Hapus sekolah"><svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg></button>' +
                    "</span>"
                  : "-") +
                "</td>",
            ].join("");
            schoolTableBody.appendChild(tr);
          });
        }
      }
      if (classTableBody) {
        if (!classItems.length) {
          classTableBody.innerHTML =
            '<tr><td colspan="5" class="muted">Belum ada data kelas.</td></tr>';
        } else {
          classTableBody.innerHTML = "";
          classItems.forEach(function (it) {
            const school =
              masterSummaryState.schoolsByID[String(it.school_id)] || {};
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(schoolLabel(school)) + "</td>",
              "<td>" + escapeHtml(String(it.grade_level || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
              "<td>" +
                (canEditMaster
                  ? '<span class="action-icons">' +
                    '<button class="icon-only-btn" type="button" data-master-entity="class" data-master-action="edit" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Ubah kelas" aria-label="Ubah kelas"><svg viewBox="0 0 24 24" focusable="false"><path d="M4 20h4l10-10-4-4L4 16v4z"/><path d="M12 6l4 4"/></svg></button>' +
                    '<button class="icon-only-btn danger" type="button" data-master-entity="class" data-master-action="delete" data-id="' +
                    escapeHtml(String(it.id || "")) +
                    '" title="Hapus kelas" aria-label="Hapus kelas"><svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg></button>' +
                    "</span>"
                  : "-") +
                "</td>",
            ].join("");
            classTableBody.appendChild(tr);
          });
        }
      }
    }

    async function loadMasterSummaryTable() {
      const levelsPromise = loadEducationLevels(isAdmin());
      const schoolsPromise = api("/api/v1/admin/schools?all=1", "GET");
      const classesPromise = api("/api/v1/admin/classes?all=1", "GET");
      const all = await Promise.all([
        levelsPromise,
        schoolsPromise,
        classesPromise,
      ]);
      const levels = Array.isArray(all[0]) ? all[0] : [];
      const schools = Array.isArray(all[1]) ? all[1] : [];
      const classes = Array.isArray(all[2]) ? all[2] : [];
      renderMasterSummaryTable(levels, schools, classes);
      return { levels: levels, schools: schools, classes: classes };
    }

    async function loadUsers() {
      const params = new URLSearchParams();
      params.set("limit", String(currentLimit));
      params.set("offset", String(currentOffset));
      if (currentRole) params.set("role", currentRole);
      if (currentQuery) params.set("q", currentQuery);
      const out = await api("/api/v1/admin/users?" + params.toString(), "GET");
      pretty(regOut, out);
      const items = Array.isArray(out.items) ? out.items : [];
      lastLoadedCount = items.length;
      text(
        regSummary,
        "Peran: " +
          (out.role || "semua") +
          " | Pencarian: " +
          (out.q || "-") +
          " | Data dimuat: " +
          String(items.length),
      );
      if (pageInfo) {
        const start = items.length ? currentOffset + 1 : currentOffset;
        const end = currentOffset + items.length;
        text(pageInfo, "Baris " + String(start) + " - " + String(end));
      }
      if (prevPageBtn) prevPageBtn.disabled = currentOffset <= 0;
      if (nextPageBtn) nextPageBtn.disabled = items.length < currentLimit;
      if (!regBody) return;
      if (!items.length) {
        regBody.innerHTML =
          '<tr><td colspan="10" class="muted">Tidak ada data pengguna.</td></tr>';
        return;
      }
      regBody.innerHTML = "";
      Object.keys(usersByID).forEach(function (k) {
        delete usersByID[k];
      });
      items.forEach(function (it) {
        usersByID[String(it.id || "")] = it;
        const tr = document.createElement("tr");
        const activeLabel = it.is_active ? "ya" : "tidak";
        const detailBtn =
          '<button class="icon-only-btn" type="button" data-action="detail" data-id="' +
          escapeHtml(String(it.id || "")) +
          '" title="Lihat detil" aria-label="Lihat detil">' +
          '<svg viewBox="0 0 24 24" focusable="false"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6z"/><circle cx="12" cy="12" r="2.5"/></svg></button>';
        const editBtn =
          '<button class="icon-only-btn" type="button" data-action="edit" data-id="' +
          escapeHtml(String(it.id || "")) +
          '" title="Ubah pengguna" aria-label="Ubah pengguna">' +
          '<svg viewBox="0 0 24 24" focusable="false"><path d="M4 20h4l10-10-4-4L4 16v4z"/><path d="M12 6l4 4"/></svg></button>';
        const disableBtn =
          '<button class="icon-only-btn danger" type="button" data-action="deactivate" data-id="' +
          escapeHtml(String(it.id || "")) +
          '" title="Nonaktifkan pengguna" aria-label="Nonaktifkan pengguna">' +
          '<svg viewBox="0 0 24 24" focusable="false"><path d="M18 6L6 18M6 6l12 12"/></svg></button>';
        const actionIcons = isAdmin()
          ? '<span class="action-icons">' +
            detailBtn +
            editBtn +
            disableBtn +
            "</span>"
          : '<span class="action-icons">' + detailBtn + "</span>";
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(it.username || "") + "</td>",
          "<td>" + escapeHtml(it.full_name || "") + "</td>",
          "<td>" + escapeHtml(it.role || "") + "</td>",
          "<td>" + escapeHtml(it.school_name || "-") + "</td>",
          "<td>" + escapeHtml(it.email || "") + "</td>",
          "<td>" + escapeHtml(it.account_status || "") + "</td>",
          "<td>" + escapeHtml(activeLabel) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.created_at)) + "</td>",
          "<td>" + actionIcons + "</td>",
        ].join("");
        regBody.appendChild(tr);
      });
    }

    async function loadDashboardStats() {
      const out = await api("/api/v1/admin/dashboard/stats", "GET");
      if (statAdmin) statAdmin.textContent = String(out.admin_count || 0);
      if (statProktor) statProktor.textContent = String(out.proktor_count || 0);
      if (statGuru) statGuru.textContent = String(out.guru_count || 0);
      if (statSiswa) statSiswa.textContent = String(out.siswa_count || 0);
      if (statSekolah) statSekolah.textContent = String(out.school_count || 0);
    }

    if (regFilterForm) {
      const triggerFilterSubmit = function () {
        if (typeof regFilterForm.requestSubmit === "function") {
          regFilterForm.requestSubmit();
          return;
        }
        regFilterForm.dispatchEvent(
          new Event("submit", { cancelable: true, bubbles: true }),
        );
      };

      regFilterForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(regFilterForm);
        currentRole = String(fd.get("role") || "").trim();
        currentQuery = String(fd.get("q") || "").trim();
        currentLimit = Number(fd.get("limit") || 50);
        currentOffset = 0;
        const done = beginBusy(
          regFilterForm,
          message,
          "Memuat daftar pengguna...",
        );
        try {
          await loadUsers();
          setMsg("Daftar pengguna berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat pengguna: " + err.message);
        } finally {
          done();
        }
      });

      const roleInput = regFilterForm.elements.namedItem("role");
      if (roleInput) {
        roleInput.addEventListener("change", function () {
          currentOffset = 0;
          triggerFilterSubmit();
        });
      }

      const qInput = regFilterForm.elements.namedItem("q");
      if (qInput) {
        qInput.addEventListener("input", function () {
          if (filterDebounceTimer) window.clearTimeout(filterDebounceTimer);
          filterDebounceTimer = window.setTimeout(function () {
            currentOffset = 0;
            triggerFilterSubmit();
          }, 350);
        });
      }
    }

    menuButtons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        const target = btn.getAttribute("data-admin-menu") || "users";
        activateMenu(target);
      });
    });

    if (prevPageBtn) {
      prevPageBtn.addEventListener("click", async function () {
        if (currentOffset <= 0) return;
        currentOffset = Math.max(0, currentOffset - currentLimit);
        const done = beginBusy(
          prevPageBtn,
          message,
          "Memuat halaman sebelumnya...",
        );
        try {
          await loadUsers();
          setMsg("Halaman sebelumnya dimuat.");
        } catch (err) {
          setMsg("Gagal pindah halaman: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (nextPageBtn) {
      nextPageBtn.addEventListener("click", async function () {
        if (lastLoadedCount < currentLimit) return;
        currentOffset += currentLimit;
        const done = beginBusy(
          nextPageBtn,
          message,
          "Memuat halaman berikutnya...",
        );
        try {
          await loadUsers();
          setMsg("Halaman berikutnya dimuat.");
        } catch (err) {
          currentOffset = Math.max(0, currentOffset - currentLimit);
          setMsg("Gagal pindah halaman: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (addUserBtn) {
      addUserBtn.hidden = !isAdmin();
      addUserBtn.addEventListener("click", function () {
        if (!isAdmin()) return;
        if (userCreateForm) {
          userCreateForm.reset();
          fillSchoolSelect(userCreateForm.elements.namedItem("school_id"));
          resetClassSelect(
            userCreateForm.elements.namedItem("class_id"),
            "Pilih Sekolah Dulu",
          );
        }
        if (
          userCreateDialog &&
          typeof userCreateDialog.showModal === "function"
        ) {
          userCreateDialog.showModal();
        }
      });
    }
    if (levelAddBtn) levelAddBtn.hidden = !isAdmin();
    if (schoolAddBtn) schoolAddBtn.hidden = !isAdmin();
    if (classAddBtn) classAddBtn.hidden = !isAdmin();

    if (exportUsersBtn) {
      exportUsersBtn.hidden = !isAdmin();
      exportUsersBtn.addEventListener("click", async function () {
        if (!isAdmin()) return;
        const done = beginBusy(
          exportUsersBtn,
          message,
          "Menyiapkan file Excel pengguna...",
        );
        try {
          await exportUsersExcelFile();
          setMsg("Ekspor pengguna selesai.");
        } catch (err) {
          setMsg("Gagal ekspor pengguna: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (importUsersBtn) {
      importUsersBtn.hidden = !isAdmin();
      importUsersBtn.addEventListener("click", function () {
        if (!isAdmin()) return;
        if (userImportForm) userImportForm.reset();
        if (userImportOut) userImportOut.textContent = "";
        if (
          userImportDialog &&
          typeof userImportDialog.showModal === "function"
        ) {
          userImportDialog.showModal();
        }
      });
    }

    if (userImportCancelBtn) {
      userImportCancelBtn.addEventListener("click", function () {
        if (userImportDialog && typeof userImportDialog.close === "function") {
          userImportDialog.close();
        }
      });
    }

    if (userImportErrorCloseBtn) {
      userImportErrorCloseBtn.addEventListener("click", function () {
        if (
          userImportErrorDialog &&
          typeof userImportErrorDialog.close === "function"
        ) {
          userImportErrorDialog.close();
        }
      });
    }

    if (userCreateCancelBtn) {
      userCreateCancelBtn.addEventListener("click", function () {
        if (userCreateDialog && typeof userCreateDialog.close === "function") {
          userCreateDialog.close();
        }
      });
    }

    if (userUpdateCancelBtn) {
      userUpdateCancelBtn.addEventListener("click", function () {
        if (userUpdateDialog && typeof userUpdateDialog.close === "function") {
          userUpdateDialog.close();
        }
      });
    }

    if (userDeleteCancelBtn) {
      userDeleteCancelBtn.addEventListener("click", function () {
        if (userDeleteDialog && typeof userDeleteDialog.close === "function") {
          userDeleteDialog.close();
        }
      });
    }

    if (regBody) {
      regBody.addEventListener("click", function (e) {
        const btn =
          e.target && e.target.closest("button[data-action][data-id]");
        if (!btn) return;
        const action = String(btn.getAttribute("data-action") || "");
        const id = String(btn.getAttribute("data-id") || "");
        const userItem = usersByID[id];
        if (!userItem) return;

        if (action === "detail") {
          if (userDetailOut)
            userDetailOut.textContent = JSON.stringify(userItem, null, 2);
          if (
            userDetailDialog &&
            typeof userDetailDialog.showModal === "function"
          ) {
            userDetailDialog.showModal();
          }
          return;
        }

        if (!isAdmin()) return;

        if (action === "edit") {
          if (userUpdateForm) {
            userUpdateForm.elements.namedItem("id").value = String(
              userItem.id || "",
            );
            userUpdateForm.elements.namedItem("full_name").value = String(
              userItem.full_name || "",
            );
            userUpdateForm.elements.namedItem("email").value = String(
              userItem.email || "",
            );
            userUpdateForm.elements.namedItem("role").value = String(
              userItem.role || "siswa",
            );
            userUpdateForm.elements.namedItem("password").value = "";
            fillSchoolSelect(
              userUpdateForm.elements.namedItem("school_id"),
              userItem.school_id,
            );
            const schoolID = Number(userItem.school_id || 0);
            const classID = Number(userItem.class_id || 0);
            const classSelect = userUpdateForm.elements.namedItem("class_id");
            if (schoolID > 0) {
              loadClassesBySchool(schoolID)
                .then(function (classes) {
                  fillClassSelect(classSelect, classes, classID);
                })
                .catch(function () {
                  resetClassSelect(classSelect, "Gagal memuat kelas");
                });
            } else {
              resetClassSelect(classSelect, "Pilih Sekolah Dulu");
            }
          }
          if (
            userUpdateDialog &&
            typeof userUpdateDialog.showModal === "function"
          ) {
            userUpdateDialog.showModal();
          }
          return;
        }

        if (action === "deactivate") {
          if (userDeleteForm)
            userDeleteForm.elements.namedItem("id").value = String(
              userItem.id || "",
            );
          if (userDeleteLabel) {
            userDeleteLabel.textContent =
              "Yakin nonaktifkan pengguna " +
              String(userItem.full_name || userItem.username || "") +
              " (ID " +
              String(userItem.id || "") +
              ")?";
          }
          if (
            userDeleteDialog &&
            typeof userDeleteDialog.showModal === "function"
          ) {
            userDeleteDialog.showModal();
          }
        }
      });
    }

    if (isAdmin() && userCreateForm) {
      const createSchoolSelect = userCreateForm.elements.namedItem("school_id");
      const createClassSelect = userCreateForm.elements.namedItem("class_id");
      if (createSchoolSelect && createClassSelect) {
        createSchoolSelect.addEventListener("change", function () {
          const sid = Number(createSchoolSelect.value || 0);
          if (sid <= 0) {
            resetClassSelect(createClassSelect, "Pilih Sekolah Dulu");
            return;
          }
          loadClassesBySchool(sid)
            .then(function (classes) {
              fillClassSelect(createClassSelect, classes);
              pickFirstClassIfAny(createClassSelect);
            })
            .catch(function () {
              resetClassSelect(createClassSelect, "Gagal memuat kelas");
            });
        });
      }

      userCreateForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(userCreateForm);
        const done = beginBusy(
          userCreateForm,
          message,
          "Menyimpan pengguna...",
        );
        try {
          const schoolID = toNullableNumber(fd.get("school_id"));
          const classID = toNullableNumber(fd.get("class_id"));
          if (schoolID !== null && classID === null) {
            setMsg("Jika memilih sekolah, kelas wajib dipilih.");
            return;
          }
          const out = await api("/api/v1/admin/users", "POST", {
            username: String(fd.get("username") || "").trim(),
            full_name: String(fd.get("full_name") || "").trim(),
            email: String(fd.get("email") || "").trim(),
            role: String(fd.get("role") || "").trim(),
            password: String(fd.get("password") || ""),
            school_id: schoolID,
            class_id: classID,
          });
          pretty(regOut, out);
          setMsg("Pengguna berhasil dibuat.");
          userCreateForm.reset();
          if (
            userCreateDialog &&
            typeof userCreateDialog.close === "function"
          ) {
            userCreateDialog.close();
          }
          await loadUsers();
          await loadDashboardStats();
        } catch (err) {
          setMsg("Gagal membuat pengguna: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && userUpdateForm) {
      const updateSchoolSelect = userUpdateForm.elements.namedItem("school_id");
      const updateClassSelect = userUpdateForm.elements.namedItem("class_id");
      if (updateSchoolSelect && updateClassSelect) {
        updateSchoolSelect.addEventListener("change", function () {
          const sid = Number(updateSchoolSelect.value || 0);
          if (sid <= 0) {
            resetClassSelect(updateClassSelect, "Pilih Sekolah Dulu");
            return;
          }
          loadClassesBySchool(sid)
            .then(function (classes) {
              fillClassSelect(updateClassSelect, classes);
              pickFirstClassIfAny(updateClassSelect);
            })
            .catch(function () {
              resetClassSelect(updateClassSelect, "Gagal memuat kelas");
            });
        });
      }

      userUpdateForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(userUpdateForm);
        const id = Number(fd.get("id") || 0);
        const done = beginBusy(
          userUpdateForm,
          message,
          "Memperbarui pengguna...",
        );
        try {
          const schoolID = toNullableNumber(fd.get("school_id"));
          const classID = toNullableNumber(fd.get("class_id"));
          if (schoolID !== null && classID === null) {
            setMsg("Jika memilih sekolah, kelas wajib dipilih.");
            return;
          }
          const out = await api("/api/v1/admin/users/" + id, "PUT", {
            full_name: String(fd.get("full_name") || "").trim(),
            email: String(fd.get("email") || "").trim(),
            role: String(fd.get("role") || "").trim(),
            password: String(fd.get("password") || ""),
            school_id: schoolID === null ? 0 : schoolID,
            class_id: classID === null ? 0 : classID,
          });
          pretty(regOut, out);
          setMsg("Pengguna berhasil diperbarui.");
          if (
            userUpdateDialog &&
            typeof userUpdateDialog.close === "function"
          ) {
            userUpdateDialog.close();
          }
          await loadUsers();
          await loadDashboardStats();
        } catch (err) {
          setMsg("Gagal memperbarui pengguna: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && userDeleteForm) {
      userDeleteForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(userDeleteForm);
        const id = Number(fd.get("id") || 0);
        const done = beginBusy(
          userDeleteForm,
          message,
          "Menonaktifkan pengguna...",
        );
        try {
          const out = await api("/api/v1/admin/users/" + id, "DELETE", {});
          pretty(regOut, out);
          setMsg("Pengguna berhasil dinonaktifkan.");
          userDeleteForm.reset();
          if (
            userDeleteDialog &&
            typeof userDeleteDialog.close === "function"
          ) {
            userDeleteDialog.close();
          }
          await loadUsers();
          await loadDashboardStats();
        } catch (err) {
          setMsg("Gagal menonaktifkan pengguna: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && userImportForm) {
      userImportForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(userImportForm);
        const file = fd.get("file");
        if (!file || !file.name) {
          setMsg("File Excel wajib dipilih.");
          return;
        }
        const done = beginBusy(
          userImportForm,
          message,
          "Mengimpor data pengguna dari Excel...",
        );
        try {
          const out = await apiMultipart(
            "/api/v1/admin/users/import",
            "POST",
            fd,
          );
          pretty(userImportOut, out);
          pretty(regOut, out);
          const report =
            out && out.report && typeof out.report === "object"
              ? out.report
              : {};
          const failed = Number(report.failed_rows || 0);
          const errors = Array.isArray(report.errors) ? report.errors : [];

          if (failed > 0) {
            const details = errors.length
              ? errors
                  .map(function (it) {
                    const row = Number(it.row || 0);
                    const userLabel = String(it.username || "").trim();
                    const errMsg = String(it.error || "error tidak diketahui");
                    if (userLabel) {
                      return (
                        "Baris " +
                        String(row) +
                        " | username: " +
                        userLabel +
                        " | " +
                        errMsg
                      );
                    }
                    return "Baris " + String(row) + " | " + errMsg;
                  })
                  .join("\n")
              : "Ada baris gagal, tetapi detail tidak tersedia.";
            if (userImportErrorOut) userImportErrorOut.textContent = details;
            if (
              userImportErrorDialog &&
              typeof userImportErrorDialog.showModal === "function"
            ) {
              userImportErrorDialog.showModal();
            }
          } else {
            userImportForm.reset();
            if (
              userImportDialog &&
              typeof userImportDialog.close === "function"
            ) {
              userImportDialog.close();
            }
          }

          setMsg(
            "Impor selesai. Berhasil: " +
              String(report.success_rows || 0) +
              ", gagal: " +
              String(report.failed_rows || 0) +
              ".",
          );
          await loadUsers();
          await loadDashboardStats();
        } catch (err) {
          if (userImportErrorOut) {
            userImportErrorOut.textContent =
              "Impor gagal:\n" + String(err.message || "Request failed");
          }
          if (
            userImportErrorDialog &&
            typeof userImportErrorDialog.showModal === "function"
          ) {
            userImportErrorDialog.showModal();
          }
          setMsg("Gagal impor pengguna: " + err.message);
        } finally {
          done();
        }
      });
    }

    const importForm = document.getElementById("admin-import-form");
    if (importForm) {
      importForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(importForm);
        const file = fd.get("file");
        if (!file || !file.name) {
          setMsg("File CSV wajib dipilih.");
          return;
        }
        const done = beginBusy(
          importForm,
          message,
          "Mengimpor data siswa CSV...",
        );
        try {
          const out = await apiMultipart(
            "/api/v1/admin/imports/students",
            "POST",
            fd,
          );
          pretty(masterOut, out);
          setMsg("Impor CSV selesai.");
          importForm.reset();
        } catch (err) {
          setMsg("Gagal impor CSV: " + err.message);
        } finally {
          done();
        }
      });
    }

    function setMasterMsg(msg) {
      if (masterMessage) masterMessage.textContent = msg;
      else setMsg(msg);
    }

    function openLevelDialog(editItem) {
      if (!levelDialogForm || !levelDialog) return;
      levelDialogForm.reset();
      levelDialogForm.elements.namedItem("id").value = editItem
        ? String(editItem.id || "")
        : "";
      levelDialogForm.elements.namedItem("name").value = editItem
        ? String(editItem.name || "")
        : "";
      if (levelDialogTitle) {
        levelDialogTitle.textContent = editItem
          ? "Ubah Jenjang"
          : "Tambah Jenjang";
      }
      if (typeof levelDialog.showModal === "function") levelDialog.showModal();
    }

    function openSchoolDialog(editItem) {
      if (!schoolDialogForm || !schoolDialog) return;
      schoolDialogForm.reset();
      schoolDialogForm.elements.namedItem("id").value = editItem
        ? String(editItem.id || "")
        : "";
      schoolDialogForm.elements.namedItem("name").value = editItem
        ? String(editItem.name || "")
        : "";
      schoolDialogForm.elements.namedItem("code").value = editItem
        ? String(editItem.code || "")
        : "";
      schoolDialogForm.elements.namedItem("address").value = editItem
        ? String(editItem.address || "")
        : "";
      if (schoolDialogTitle) {
        schoolDialogTitle.textContent = editItem
          ? "Ubah Sekolah"
          : "Tambah Sekolah";
      }
      if (typeof schoolDialog.showModal === "function")
        schoolDialog.showModal();
    }

    function openClassDialog(editItem) {
      if (!classDialogForm || !classDialog) return;
      classDialogForm.reset();
      const schoolSelect = classDialogForm.elements.namedItem("school_id");
      fillSchoolSelectRequired(schoolSelect, editItem ? editItem.school_id : 0);
      classDialogForm.elements.namedItem("id").value = editItem
        ? String(editItem.id || "")
        : "";
      classDialogForm.elements.namedItem("grade_level").value = editItem
        ? String(editItem.grade_level || "")
        : "";
      classDialogForm.elements.namedItem("name").value = editItem
        ? String(editItem.name || "")
        : "";
      if (classDialogTitle) {
        classDialogTitle.textContent = editItem ? "Ubah Kelas" : "Tambah Kelas";
      }
      if (typeof classDialog.showModal === "function") classDialog.showModal();
    }

    function openDeleteDialog(entity, id, label) {
      if (!masterDeleteForm || !masterDeleteDialog) return;
      masterDeleteForm.reset();
      masterDeleteForm.elements.namedItem("entity").value = entity;
      masterDeleteForm.elements.namedItem("id").value = String(id);
      if (masterDeleteLabel) masterDeleteLabel.textContent = label;
      if (typeof masterDeleteDialog.showModal === "function")
        masterDeleteDialog.showModal();
    }

    if (levelAddBtn && isAdmin()) {
      levelAddBtn.addEventListener("click", function () {
        openLevelDialog(null);
      });
    }
    if (schoolAddBtn && isAdmin()) {
      schoolAddBtn.addEventListener("click", function () {
        openSchoolDialog(null);
      });
    }
    if (classAddBtn && isAdmin()) {
      classAddBtn.addEventListener("click", function () {
        openClassDialog(null);
      });
    }

    if (levelDialogCancelBtn) {
      levelDialogCancelBtn.addEventListener("click", function () {
        if (levelDialog && typeof levelDialog.close === "function") {
          levelDialog.close();
        }
      });
    }
    if (schoolDialogCancelBtn) {
      schoolDialogCancelBtn.addEventListener("click", function () {
        if (schoolDialog && typeof schoolDialog.close === "function") {
          schoolDialog.close();
        }
      });
    }
    if (classDialogCancelBtn) {
      classDialogCancelBtn.addEventListener("click", function () {
        if (classDialog && typeof classDialog.close === "function") {
          classDialog.close();
        }
      });
    }
    if (masterDeleteCancelBtn) {
      masterDeleteCancelBtn.addEventListener("click", function () {
        if (
          masterDeleteDialog &&
          typeof masterDeleteDialog.close === "function"
        ) {
          masterDeleteDialog.close();
        }
      });
    }

    if (masterRefreshBtn) {
      masterRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          masterRefreshBtn,
          message,
          "Memuat ulang data master...",
        );
        try {
          await loadMasterSummaryTable();
          setMasterMsg("Data master berhasil dimuat.");
        } catch (err) {
          setMasterMsg("Gagal memuat data master: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (levelTableBody) {
      levelTableBody.addEventListener("click", function (e) {
        const btn =
          e.target &&
          e.target.closest(
            "button[data-master-entity][data-master-action][data-id]",
          );
        if (!btn || !isAdmin()) return;
        const id = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-master-action") || "");
        const item = masterSummaryState.levelsByID[String(id)];
        if (!id || !item) return;
        if (action === "edit") {
          openLevelDialog(item);
        } else if (action === "delete") {
          openDeleteDialog(
            "level",
            id,
            "Hapus jenjang " + String(item.name || "") + "?",
          );
        }
      });
    }

    if (schoolTableBody) {
      schoolTableBody.addEventListener("click", function (e) {
        const btn =
          e.target &&
          e.target.closest(
            "button[data-master-entity][data-master-action][data-id]",
          );
        if (!btn || !isAdmin()) return;
        const id = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-master-action") || "");
        const item = masterSummaryState.schoolsByID[String(id)];
        if (!id || !item) return;
        if (action === "edit") {
          openSchoolDialog(item);
        } else if (action === "delete") {
          openDeleteDialog(
            "school",
            id,
            "Hapus sekolah " + String(item.name || "") + "?",
          );
        }
      });
    }

    if (classTableBody) {
      classTableBody.addEventListener("click", function (e) {
        const btn =
          e.target &&
          e.target.closest(
            "button[data-master-entity][data-master-action][data-id]",
          );
        if (!btn || !isAdmin()) return;
        const id = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-master-action") || "");
        const item = masterSummaryState.classesByID[String(id)];
        if (!id || !item) return;
        if (action === "edit") {
          openClassDialog(item);
        } else if (action === "delete") {
          openDeleteDialog(
            "class",
            id,
            "Hapus kelas " + String(item.name || "") + "?",
          );
        }
      });
    }

    if (isAdmin() && levelDialogForm) {
      levelDialogForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(levelDialogForm);
        const id = Number(fd.get("id") || 0);
        const name = String(fd.get("name") || "").trim();
        const done = beginBusy(
          levelDialogForm,
          message,
          "Menyimpan jenjang...",
        );
        try {
          if (id > 0) {
            await api("/api/v1/admin/levels/" + id, "PUT", { name: name });
            setMasterMsg("Jenjang berhasil diperbarui.");
          } else {
            await api("/api/v1/admin/levels", "POST", { name: name });
            setMasterMsg("Jenjang berhasil ditambahkan.");
          }
          if (levelDialog && typeof levelDialog.close === "function") {
            levelDialog.close();
          }
          await loadMasterSummaryTable();
        } catch (err) {
          setMasterMsg("Simpan jenjang gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && schoolDialogForm) {
      schoolDialogForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(schoolDialogForm);
        const id = Number(fd.get("id") || 0);
        const payload = {
          name: String(fd.get("name") || "").trim(),
          code: String(fd.get("code") || "").trim(),
          address: String(fd.get("address") || "").trim(),
        };
        const done = beginBusy(
          schoolDialogForm,
          message,
          "Menyimpan sekolah...",
        );
        try {
          if (id > 0) {
            await api("/api/v1/admin/schools/" + id, "PUT", payload);
            setMasterMsg("Sekolah berhasil diperbarui.");
          } else {
            await api("/api/v1/admin/schools", "POST", payload);
            setMasterMsg("Sekolah berhasil ditambahkan.");
          }
          if (schoolDialog && typeof schoolDialog.close === "function") {
            schoolDialog.close();
          }
          await loadMasterSummaryTable();
          await loadDashboardStats();
          await loadSchoolsForUserForms();
        } catch (err) {
          setMasterMsg("Simpan sekolah gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && classDialogForm) {
      classDialogForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(classDialogForm);
        const id = Number(fd.get("id") || 0);
        const payload = {
          school_id: Number(fd.get("school_id") || 0),
          grade_level: String(fd.get("grade_level") || "").trim(),
          name: String(fd.get("name") || "").trim(),
        };
        const done = beginBusy(classDialogForm, message, "Menyimpan kelas...");
        try {
          if (id > 0) {
            await api("/api/v1/admin/classes/" + id, "PUT", payload);
            setMasterMsg("Kelas berhasil diperbarui.");
          } else {
            await api("/api/v1/admin/classes", "POST", payload);
            setMasterMsg("Kelas berhasil ditambahkan.");
          }
          if (classDialog && typeof classDialog.close === "function") {
            classDialog.close();
          }
          await loadMasterSummaryTable();
        } catch (err) {
          setMasterMsg("Simpan kelas gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (isAdmin() && masterDeleteForm) {
      masterDeleteForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(masterDeleteForm);
        const entity = String(fd.get("entity") || "");
        const id = Number(fd.get("id") || 0);
        if (!entity || !id) return;
        const done = beginBusy(masterDeleteForm, message, "Menghapus data...");
        try {
          if (entity === "level") {
            await api("/api/v1/admin/levels/" + id, "DELETE", {});
            setMasterMsg("Jenjang berhasil dihapus.");
          } else if (entity === "school") {
            await api("/api/v1/admin/schools/" + id, "DELETE", {});
            setMasterMsg("Sekolah berhasil dihapus.");
          } else if (entity === "class") {
            await api("/api/v1/admin/classes/" + id, "DELETE", {});
            setMasterMsg("Kelas berhasil dihapus.");
          }
          if (
            masterDeleteDialog &&
            typeof masterDeleteDialog.close === "function"
          ) {
            masterDeleteDialog.close();
          }
          await loadMasterSummaryTable();
          await loadDashboardStats();
          await loadSchoolsForUserForms();
          await loadUsers();
        } catch (err) {
          setMasterMsg("Hapus data gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    try {
      if (regFilterForm) {
        const fd = new FormData(regFilterForm);
        currentRole = String(fd.get("role") || "").trim();
        currentQuery = String(fd.get("q") || "").trim();
        currentLimit = Number(fd.get("limit") || 50);
      }
      await loadSchoolsForUserForms();
      if (userCreateForm) {
        fillSchoolSelect(userCreateForm.elements.namedItem("school_id"));
        resetClassSelect(
          userCreateForm.elements.namedItem("class_id"),
          "Pilih Sekolah Dulu",
        );
      }
      if (userUpdateForm) {
        fillSchoolSelect(userUpdateForm.elements.namedItem("school_id"));
        resetClassSelect(
          userUpdateForm.elements.namedItem("class_id"),
          "Pilih Sekolah Dulu",
        );
      }
      await loadUsers();
      await loadDashboardStats();
      await loadMasterSummaryTable();
      activateMenu("users");
      setMsg("Dashboard admin siap.");
    } catch (err) {
      setMsg("Gagal memuat data awal admin: " + err.message);
    }
  }

  async function initAuthoringPage() {
    const root = document.querySelector('[data-page="authoring_content"]');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }
    if (!["admin", "proktor", "guru"].includes(String(user.role || ""))) {
      alert("Halaman authoring hanya untuk admin/proktor/guru");
      window.location.href = "/";
      return;
    }

    const message = document.getElementById("authoring-message");
    const stimulusOut = document.getElementById("stimulus-output");
    const versionOut = document.getElementById("version-output");
    const parallelOut = document.getElementById("parallel-output");
    const reviewOut = document.getElementById("review-output");
    const reviewTaskList = document.getElementById("review-task-list");

    function pretty(el, data) {
      if (el) el.textContent = JSON.stringify(data, null, 2);
    }
    function setMsg(msg) {
      text(message, msg);
    }

    const stimulusForm = document.getElementById("stimulus-form");
    if (stimulusForm) {
      stimulusForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusForm);
        const done = beginBusy(stimulusForm, message, "Menyimpan stimulus...");
        try {
          const out = await api("/api/v1/stimuli", "POST", {
            subject_id: Number(fd.get("subject_id") || 0),
            title: String(fd.get("title") || ""),
            stimulus_type: String(fd.get("stimulus_type") || ""),
            content: JSON.parse(String(fd.get("content_raw") || "{}")),
          });
          pretty(stimulusOut, out);
          setMsg("Stimulus berhasil disimpan.");
        } catch (err) {
          setMsg("Gagal simpan stimulus: " + err.message);
        } finally {
          done();
        }
      });
    }

    const stimulusListForm = document.getElementById("stimulus-list-form");
    if (stimulusListForm) {
      stimulusListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusListForm);
        const done = beginBusy(
          stimulusListForm,
          message,
          "Memuat daftar stimulus...",
        );
        try {
          const out = await api(
            "/api/v1/stimuli?subject_id=" +
              encodeURIComponent(String(fd.get("subject_id") || "")),
            "GET",
          );
          pretty(stimulusOut, out);
          setMsg("List stimulus berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal list stimulus: " + err.message);
        } finally {
          done();
        }
      });
    }

    const versionCreateForm = document.getElementById("version-create-form");
    if (versionCreateForm) {
      versionCreateForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(versionCreateForm);
        const done = beginBusy(
          versionCreateForm,
          message,
          "Menyimpan draft versi soal...",
        );
        try {
          const qid = Number(fd.get("question_id") || 0);
          const payload = {
            stem_html: String(fd.get("stem_html") || ""),
            explanation_html: String(fd.get("explanation_html") || ""),
            hint_html: String(fd.get("hint_html") || ""),
            answer_key: JSON.parse(String(fd.get("answer_key_raw") || "{}")),
          };
          const stimulusID = Number(fd.get("stimulus_id") || 0);
          if (stimulusID > 0) payload.stimulus_id = stimulusID;
          const durationRaw = String(fd.get("duration_seconds") || "").trim();
          if (durationRaw !== "")
            payload.duration_seconds = Number(durationRaw);
          const weightRaw = String(fd.get("weight") || "").trim();
          if (weightRaw !== "") payload.weight = Number(weightRaw);
          const changeNote = String(fd.get("change_note") || "").trim();
          if (changeNote) payload.change_note = changeNote;

          const out = await api(
            "/api/v1/questions/" + qid + "/versions",
            "POST",
            payload,
          );
          pretty(versionOut, out);
          setMsg("Draft versi soal berhasil disimpan.");
        } catch (err) {
          setMsg("Gagal simpan versi: " + err.message);
        } finally {
          done();
        }
      });
    }

    const versionFinalizeForm = document.getElementById(
      "version-finalize-form",
    );
    if (versionFinalizeForm) {
      versionFinalizeForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(versionFinalizeForm);
        const done = beginBusy(
          versionFinalizeForm,
          message,
          "Memfinalisasi versi soal...",
        );
        try {
          const qid = Number(fd.get("question_id") || 0);
          const ver = Number(fd.get("version_no") || 0);
          const out = await api(
            "/api/v1/questions/" + qid + "/versions/" + ver + "/finalize",
            "POST",
          );
          pretty(versionOut, out);
          setMsg("Versi berhasil difinalkan.");
        } catch (err) {
          setMsg("Gagal finalize versi: " + err.message);
        } finally {
          done();
        }
      });
    }

    const versionListForm = document.getElementById("version-list-form");
    if (versionListForm) {
      versionListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(versionListForm);
        const done = beginBusy(
          versionListForm,
          message,
          "Memuat daftar versi soal...",
        );
        try {
          const qid = Number(fd.get("question_id") || 0);
          const out = await api(
            "/api/v1/questions/" + qid + "/versions",
            "GET",
          );
          pretty(versionOut, out);
          setMsg("List versi berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal list versi: " + err.message);
        } finally {
          done();
        }
      });
    }

    const parallelCreateForm = document.getElementById("parallel-create-form");
    if (parallelCreateForm) {
      parallelCreateForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(parallelCreateForm);
        const done = beginBusy(
          parallelCreateForm,
          message,
          "Menambahkan parallel exam...",
        );
        try {
          const examID = Number(fd.get("exam_id") || 0);
          const out = await api(
            "/api/v1/exams/" + examID + "/parallels",
            "POST",
            {
              question_id: Number(fd.get("question_id") || 0),
              parallel_group: String(fd.get("parallel_group") || "default"),
              parallel_order: Number(fd.get("parallel_order") || 0),
              parallel_label: String(fd.get("parallel_label") || ""),
            },
          );
          pretty(parallelOut, out);
          setMsg("Parallel berhasil ditambahkan.");
        } catch (err) {
          setMsg("Gagal tambah parallel: " + err.message);
        } finally {
          done();
        }
      });
    }

    const parallelListForm = document.getElementById("parallel-list-form");
    if (parallelListForm) {
      parallelListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(parallelListForm);
        const done = beginBusy(
          parallelListForm,
          message,
          "Memuat daftar parallel exam...",
        );
        try {
          const examID = Number(fd.get("exam_id") || 0);
          const group = String(fd.get("parallel_group") || "").trim();
          const qs = group
            ? "?parallel_group=" + encodeURIComponent(group)
            : "";
          const out = await api(
            "/api/v1/exams/" + examID + "/parallels" + qs,
            "GET",
          );
          pretty(parallelOut, out);
          setMsg("List parallel berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal list parallel: " + err.message);
        } finally {
          done();
        }
      });
    }

    function formatLocalDate(ts) {
      if (!ts) return "-";
      const d = new Date(ts);
      if (Number.isNaN(d.getTime())) return String(ts);
      return d.toLocaleString("id-ID");
    }

    function renderReviewTasks(items) {
      if (!reviewTaskList) return;
      if (!Array.isArray(items) || items.length === 0) {
        reviewTaskList.innerHTML =
          '<p class="muted">Belum ada review task untuk filter ini.</p>';
        return;
      }

      const blocks = items.map(function (it) {
        const note = it.note ? escapeHtml(String(it.note)) : "-";
        const examID =
          it.exam_id === null || typeof it.exam_id === "undefined"
            ? "-"
            : escapeHtml(String(it.exam_id));
        return [
          '<article class="review-item">',
          "<div><strong>Task #" +
            escapeHtml(String(it.id)) +
            "</strong> | status: <strong>" +
            escapeHtml(String(it.status || "")) +
            "</strong></div>",
          "<div>QID: " +
            escapeHtml(String(it.question_id || "")) +
            " | QVID: " +
            escapeHtml(String(it.question_version_id || "")) +
            " | reviewer: " +
            escapeHtml(String(it.reviewer_id || "")) +
            " | exam: " +
            examID +
            "</div>",
          "<div>Assigned: " +
            escapeHtml(formatLocalDate(it.assigned_at)) +
            " | Reviewed: " +
            escapeHtml(formatLocalDate(it.reviewed_at)) +
            "</div>",
          "<div>Note: " + note + "</div>",
          "</article>",
        ].join("");
      });
      reviewTaskList.innerHTML =
        '<div class="review-list">' + blocks.join("") + "</div>";
    }

    const reviewCreateForm = document.getElementById("review-create-form");
    if (reviewCreateForm) {
      reviewCreateForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(reviewCreateForm);
        const done = beginBusy(
          reviewCreateForm,
          message,
          "Membuat review task...",
        );
        try {
          const payload = {
            question_version_id: Number(fd.get("question_version_id") || 0),
            reviewer_id: Number(fd.get("reviewer_id") || 0),
            note: String(fd.get("note") || "").trim(),
          };
          const examRaw = String(fd.get("exam_id") || "").trim();
          if (examRaw !== "") payload.exam_id = Number(examRaw);

          const out = await api("/api/v1/reviews/tasks", "POST", payload);
          pretty(reviewOut, out);
          setMsg("Review task berhasil dibuat.");
        } catch (err) {
          setMsg("Gagal buat review task: " + err.message);
        } finally {
          done();
        }
      });
    }

    const reviewListForm = document.getElementById("review-list-form");
    if (reviewListForm) {
      reviewListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(reviewListForm);
        const done = beginBusy(
          reviewListForm,
          message,
          "Memuat daftar review task...",
        );
        try {
          const params = new URLSearchParams();
          const status = String(fd.get("status") || "").trim();
          const reviewerID = String(fd.get("reviewer_id") || "").trim();
          if (status) params.set("status", status);
          if (reviewerID) params.set("reviewer_id", reviewerID);
          const qs = params.toString() ? "?" + params.toString() : "";

          const out = await api("/api/v1/reviews/tasks" + qs, "GET");
          renderReviewTasks(out);
          pretty(reviewOut, out);
          setMsg("Review task berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal muat review task: " + err.message);
        } finally {
          done();
        }
      });
    }

    const decisionForm = document.getElementById("review-decision-form");
    const decisionStatus = document.getElementById("review-decision-status");
    const decisionNoteInput = document.getElementById("review-note-input");
    if (decisionStatus && decisionNoteInput) {
      const syncDecisionRule = function () {
        const mustFill = decisionStatus.value === "perlu_revisi";
        decisionNoteInput.required = mustFill;
      };
      syncDecisionRule();
      decisionStatus.addEventListener("change", syncDecisionRule);
    }
    if (decisionForm) {
      decisionForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(decisionForm);
        const done = beginBusy(
          decisionForm,
          message,
          "Menyimpan keputusan review...",
        );
        try {
          const taskID = Number(fd.get("task_id") || 0);
          const status = String(fd.get("status") || "").trim();
          const note = String(fd.get("note") || "").trim();
          if (status === "perlu_revisi" && !note) {
            setMsg("Alasan wajib diisi jika status perlu_revisi.");
            return;
          }

          const out = await api(
            "/api/v1/reviews/tasks/" + taskID + "/decision",
            "POST",
            { status: status, note: note },
          );
          pretty(reviewOut, out);
          setMsg("Keputusan review berhasil disimpan.");
        } catch (err) {
          setMsg("Gagal simpan keputusan review: " + err.message);
        } finally {
          done();
        }
      });
    }

    const reviewHistoryForm = document.getElementById("review-history-form");
    if (reviewHistoryForm) {
      reviewHistoryForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(reviewHistoryForm);
        const done = beginBusy(
          reviewHistoryForm,
          message,
          "Memuat riwayat review soal...",
        );
        try {
          const questionID = Number(fd.get("question_id") || 0);
          const out = await api(
            "/api/v1/questions/" + questionID + "/reviews",
            "GET",
          );
          pretty(reviewOut, out);
          setMsg("Riwayat review soal berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal muat riwayat review soal: " + err.message);
        } finally {
          done();
        }
      });
    }
  }

  initTopbarAuth();
  initLoginPage();
  initSimulasiPage();
  initAdminPage();
  initAuthoringPage();
  initAttemptPage();
  initResultPage();
})();
