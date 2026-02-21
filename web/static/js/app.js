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

    function homeByRole(userLike) {
      const role = String((userLike && userLike.role) || "")
        .trim()
        .toLowerCase();
      if (role === "admin") return "/admin";
      if (role === "proktor") return "/proktor";
      if (role === "guru") return "/guru";
      return "/ujian";
    }

    const user = await meOrNull();
    if (user) {
      window.location.href = homeByRole(user);
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
        const me = await meOrNull();
        window.location.href = homeByRole(me);
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
    const examTokenDialog = document.getElementById("exam-token-dialog");
    const examTokenDialogForm = document.getElementById(
      "exam-token-dialog-form",
    );
    const examTokenDialogHelp = document.getElementById(
      "exam-token-dialog-help",
    );
    const examTokenDialogInput = document.getElementById(
      "exam-token-dialog-input",
    );
    const examTokenDialogCancelBtn = document.getElementById(
      "exam-token-dialog-cancel-btn",
    );
    const studentProfileDialog = document.getElementById(
      "student-profile-dialog",
    );
    const studentProfileDialogForm = document.getElementById(
      "student-profile-dialog-form",
    );
    const studentProfileWrongBtn = document.getElementById(
      "student-profile-wrong-btn",
    );
    const profileParticipantNo = document.getElementById(
      "profile-participant-no",
    );
    const profileFullName = document.getElementById("profile-full-name");
    const profileClassName = document.getElementById("profile-class-name");
    const profileNISN = document.getElementById("profile-nisn");
    const profileSchoolName = document.getElementById("profile-school-name");
    const profileSchoolCode = document.getElementById("profile-school-code");
    const startBtn = document.getElementById("start-btn");
    let pendingStartExamID = 0;

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
          const examID = Number((x && x.id) || 0);
          const o = document.createElement("option");
          o.value = String(examID);
          o.textContent = String(x.code || "") + " - " + String(x.title || "");
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

    async function startAttemptByExamID(examID, token, busyTarget) {
      const done = beginBusy(busyTarget || form, msg, "Menyiapkan attempt...");
      try {
        const payload = { exam_id: Number(examID || 0) };
        const examToken = String(token || "").trim();
        payload.exam_token = examToken;
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
    }

    async function loadStudentProfile() {
      return api("/api/v1/auth/me/student-profile", "GET");
    }

    function setProfileCell(el, value) {
      if (!el) return;
      const raw = String(value || "").trim();
      el.textContent = raw || "-";
    }

    form.addEventListener("submit", async function (e) {
      e.preventDefault();
      const examID = Number(examSelect.value || 0);
      if (!examID) return;
      pendingStartExamID = examID;

      try {
        const profile = await loadStudentProfile();
        setProfileCell(profileParticipantNo, profile.participant_no);
        setProfileCell(profileFullName, profile.full_name);
        setProfileCell(profileClassName, profile.class_name);
        setProfileCell(profileNISN, profile.nisn);
        setProfileCell(profileSchoolName, profile.school_name);
        setProfileCell(profileSchoolCode, profile.school_code);
      } catch (err) {
        text(msg, "Gagal memuat data peserta: " + err.message);
        return;
      }

      if (
        studentProfileDialog &&
        typeof studentProfileDialog.showModal === "function"
      ) {
        studentProfileDialog.showModal();
        return;
      }
      text(
        msg,
        "Dialog konfirmasi data tidak tersedia. Hubungi proktor untuk bantuan.",
      );
    });

    if (examTokenDialogCancelBtn) {
      examTokenDialogCancelBtn.addEventListener("click", function () {
        pendingStartExamID = 0;
        if (examTokenDialog && typeof examTokenDialog.close === "function") {
          examTokenDialog.close();
        }
      });
    }

    if (studentProfileWrongBtn) {
      studentProfileWrongBtn.addEventListener("click", function () {
        pendingStartExamID = 0;
        if (
          studentProfileDialog &&
          typeof studentProfileDialog.close === "function"
        ) {
          studentProfileDialog.close();
        }
        text(msg, "Data tidak sesuai. Hubungi proktor sebelum lanjut ujian.");
      });
    }

    if (studentProfileDialogForm) {
      studentProfileDialogForm.addEventListener("submit", function (e) {
        e.preventDefault();
        if (
          studentProfileDialog &&
          typeof studentProfileDialog.close === "function"
        ) {
          studentProfileDialog.close();
        }
        if (examTokenDialogInput) examTokenDialogInput.value = "";
        if (
          examTokenDialog &&
          typeof examTokenDialog.showModal === "function"
        ) {
          examTokenDialog.showModal();
          return;
        }
        text(msg, "Dialog token tidak tersedia. Hubungi proktor.");
      });
    }

    if (examTokenDialogForm) {
      examTokenDialogForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const examID = Number(pendingStartExamID || 0);
        if (examID <= 0) return;
        const token = String(
          (examTokenDialogInput && examTokenDialogInput.value) || "",
        ).trim();
        if (!token) {
          text(msg, "Token ujian wajib diisi.");
          return;
        }
        if (examTokenDialog && typeof examTokenDialog.close === "function") {
          examTokenDialog.close();
        }
        await startAttemptByExamID(examID, token, examTokenDialogForm);
      });
    }
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
    const infoSoalDialog = document.getElementById("info-soal-dialog");
    const infoSoalContent = document.getElementById("info-soal-content");
    const infoSoalCloseBtn = document.getElementById("info-soal-close-btn");
    const daftarSoalBtn = document.getElementById("daftar-soal-btn");
    const daftarSoalPanel = document.getElementById("daftar-soal-panel");
    const daftarSoalBody = document.getElementById("daftar-soal-body");
    const fontSmallBtn = document.getElementById("font-small-btn");
    const fontMediumBtn = document.getElementById("font-medium-btn");
    const fontLargeBtn = document.getElementById("font-large-btn");
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
    const fontStorageKey = "cbtlms_attempt_font_size";

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

    function applyFontSize(size) {
      const allowed = ["small", "medium", "large"];
      const value = allowed.includes(String(size || ""))
        ? String(size)
        : "medium";
      root.setAttribute("data-font-size", value);
      if (fontSmallBtn) {
        const active = value === "small";
        fontSmallBtn.classList.toggle("active", active);
        fontSmallBtn.setAttribute("aria-pressed", active ? "true" : "false");
      }
      if (fontMediumBtn) {
        const active = value === "medium";
        fontMediumBtn.classList.toggle("active", active);
        fontMediumBtn.setAttribute("aria-pressed", active ? "true" : "false");
      }
      if (fontLargeBtn) {
        const active = value === "large";
        fontLargeBtn.classList.toggle("active", active);
        fontLargeBtn.setAttribute("aria-pressed", active ? "true" : "false");
      }
      try {
        window.localStorage.setItem(fontStorageKey, value);
      } catch (_) {
        // Ignore if storage is unavailable.
      }
    }

    function loadSavedFontSize() {
      try {
        const saved = window.localStorage.getItem(fontStorageKey);
        if (saved) return saved;
      } catch (_) {
        // Ignore if storage is unavailable.
      }
      return "medium";
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

    function questionTypeLabel(qType) {
      const v = String(qType || "").toLowerCase();
      if (v === "pg_tunggal") return "Pilihan Ganda Tunggal";
      if (v === "multi_jawaban") return "Pilihan Ganda Multi Jawaban";
      if (v === "benar_salah_pernyataan") return "Benar/Salah Pernyataan";
      return v || "-";
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
        const qState = questionState[currentNo] || {};
        const currentType = questionTypeLabel(
          currentQuestion && currentQuestion.question_type,
        );
        const remainText = String(
          (remainingLabel && remainingLabel.textContent) || "-",
        ).trim();
        const infoHTML = [
          '<div class="attempt-info-grid">',
          '<article class="attempt-info-item">' +
            '<div class="attempt-info-k">Posisi Soal</div>' +
            '<div class="attempt-info-v">' +
            escapeHtml(String(currentNo)) +
            " / " +
            escapeHtml(String(totalQuestions)) +
            "</div>" +
            "</article>",
          '<article class="attempt-info-item">' +
            '<div class="attempt-info-k">Tipe Soal</div>' +
            '<div class="attempt-info-v">' +
            escapeHtml(currentType) +
            "</div>" +
            "</article>",
          '<article class="attempt-info-item">' +
            '<div class="attempt-info-k">Status Jawaban</div>' +
            '<div class="attempt-info-v">' +
            (qState.answered ? "Sudah terjawab" : "Belum terjawab") +
            "</div>" +
            "</article>",
          '<article class="attempt-info-item">' +
            '<div class="attempt-info-k">Status Ragu-ragu</div>' +
            '<div class="attempt-info-v">' +
            (qState.doubt ? "Ya" : "Tidak") +
            "</div>" +
            "</article>",
          '<article class="attempt-info-item">' +
            '<div class="attempt-info-k">Sisa Waktu</div>' +
            '<div class="attempt-info-v">' +
            escapeHtml(remainText) +
            "</div>" +
            "</article>",
          "</div>",
          '<div class="attempt-info-note"><strong>Panduan:</strong> klik <em>Daftar Soal</em> untuk lompat nomor, jawaban tersimpan otomatis.</div>',
        ].join("");
        if (infoSoalContent) {
          html(infoSoalContent, infoHTML);
        }
        if (infoSoalDialog && typeof infoSoalDialog.showModal === "function") {
          infoSoalDialog.showModal();
          return;
        }
        text(
          message,
          "Soal saat ini " +
            currentNo +
            "/" +
            totalQuestions +
            ". Tipe: " +
            currentType +
            ".",
        );
      });
    }

    if (infoSoalCloseBtn) {
      infoSoalCloseBtn.addEventListener("click", function () {
        if (infoSoalDialog && typeof infoSoalDialog.close === "function") {
          infoSoalDialog.close();
        }
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
        if (!targetNo || targetNo === currentNo) return;
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

    if (fontSmallBtn) {
      fontSmallBtn.addEventListener("click", function () {
        applyFontSize("small");
      });
    }
    if (fontMediumBtn) {
      fontMediumBtn.addEventListener("click", function () {
        applyFontSize("medium");
      });
    }
    if (fontLargeBtn) {
      fontLargeBtn.addEventListener("click", function () {
        applyFontSize("large");
      });
    }

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
      applyFontSize(loadSavedFontSize());
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

  function initHomePage() {
    const root = document.querySelector('[data-page="home_content"]');
    if (!root) return;

    const modal = document.getElementById("home-ai-modal");
    const openBtn = document.getElementById("home-ai-open");
    const fabBtn = document.getElementById("home-ai-fab");
    const closeBtn = document.getElementById("home-ai-close");
    const sendBtn = document.getElementById("home-ai-send");
    const input = document.getElementById("home-ai-input");
    const messages = document.getElementById("home-ai-messages");

    if (!modal || !sendBtn || !input || !messages) return;

    function setOpen(open) {
      modal.classList.toggle("is-open", open);
      modal.setAttribute("aria-hidden", open ? "false" : "true");
      if (open) input.focus();
      if (openBtn)
        openBtn.setAttribute("aria-expanded", open ? "true" : "false");
    }

    function appendMessage(textValue, user) {
      const item = document.createElement("div");
      item.className =
        "home-ai-msg " + (user ? "home-ai-msg-user" : "home-ai-msg-bot");
      item.textContent = textValue;
      messages.appendChild(item);
      messages.scrollTop = messages.scrollHeight;
    }

    async function send() {
      const q = String(input.value || "").trim();
      if (!q) return;
      appendMessage(q, true);
      input.value = "";
      sendBtn.disabled = true;
      appendMessage("Sedang memproses...", false);
      const loadingEl = messages.lastElementChild;
      try {
        const res = await fetch("/api/v1/assistant", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ query: q }),
        });
        const json = await res.json().catch(function () {
          return {};
        });
        if (loadingEl && loadingEl.parentNode)
          loadingEl.parentNode.removeChild(loadingEl);
        if (!res.ok || !json.ok || !json.data || !json.data.reply) {
          throw new Error(
            extractAPIError(json, res.statusText || "Request failed"),
          );
        }
        appendMessage(String(json.data.reply), false);
      } catch (err) {
        if (loadingEl && loadingEl.parentNode)
          loadingEl.parentNode.removeChild(loadingEl);
        appendMessage(
          "Asisten sedang tidak tersedia. Coba beberapa saat lagi.",
          false,
        );
      } finally {
        sendBtn.disabled = false;
      }
    }

    if (openBtn) {
      openBtn.addEventListener("click", function () {
        setOpen(!modal.classList.contains("is-open"));
      });
    }
    if (fabBtn) {
      fabBtn.addEventListener("click", function () {
        setOpen(!modal.classList.contains("is-open"));
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener("click", function () {
        setOpen(false);
      });
    }
    sendBtn.addEventListener("click", send);
    input.addEventListener("keydown", function (event) {
      if (event.key === "Enter") {
        event.preventDefault();
        send();
      }
    });
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
    const userImportTemplateBtn = document.getElementById(
      "admin-user-import-template-btn",
    );
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
    const dashboardPanel = document.getElementById("admin-dashboard-panel");
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

    function isStudentRole(role) {
      return (
        String(role || "")
          .trim()
          .toLowerCase() === "siswa"
      );
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

    function applyUserFormRoleRequirements(formEl) {
      if (!formEl) return;
      const roleEl = formEl.elements.namedItem("role");
      const schoolEl = formEl.elements.namedItem("school_id");
      const classEl = formEl.elements.namedItem("class_id");
      const participantNoEl = formEl.elements.namedItem("participant_no");
      const nisnEl = formEl.elements.namedItem("nisn");
      if (!roleEl || !schoolEl || !classEl) return;

      const mustHaveClass = isStudentRole(roleEl.value);
      schoolEl.required = mustHaveClass;
      classEl.required = mustHaveClass;
      if (participantNoEl) participantNoEl.required = mustHaveClass;
      if (nisnEl) nisnEl.required = mustHaveClass;

      const schoolLabel = schoolEl.closest("label");
      const classLabel = classEl.closest("label");
      const participantNoLabel = participantNoEl
        ? participantNoEl.closest("label")
        : null;
      const nisnLabel = nisnEl ? nisnEl.closest("label") : null;
      if (schoolLabel) {
        schoolLabel.dataset.requiredRole = mustHaveClass ? "siswa" : "";
      }
      if (classLabel) {
        classLabel.dataset.requiredRole = mustHaveClass ? "siswa" : "";
      }
      if (participantNoLabel) {
        participantNoLabel.dataset.requiredRole = mustHaveClass ? "siswa" : "";
      }
      if (nisnLabel) {
        nisnLabel.dataset.requiredRole = mustHaveClass ? "siswa" : "";
      }
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

    async function downloadImportTemplateExcel() {
      const res = await fetch("/api/v1/admin/users/import-template", {
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
        "template_import_pengguna.xlsx",
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
      if (dashboardPanel) dashboardPanel.hidden = target !== "dashboard";
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
        const target = btn.getAttribute("data-admin-menu") || "dashboard";
        activateMenu(target);
      });
    });
    root.querySelectorAll("[data-admin-jump]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        const target = btn.getAttribute("data-admin-jump") || "dashboard";
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
          applyUserFormRoleRequirements(userCreateForm);
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

    if (userImportTemplateBtn) {
      userImportTemplateBtn.hidden = !isAdmin();
      userImportTemplateBtn.addEventListener("click", async function () {
        if (!isAdmin()) return;
        const done = beginBusy(
          userImportTemplateBtn,
          message,
          "Menyiapkan template import pengguna...",
        );
        try {
          await downloadImportTemplateExcel();
          setMsg("Template import berhasil diunduh.");
        } catch (err) {
          setMsg("Gagal mengunduh template import: " + err.message);
        } finally {
          done();
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
            userUpdateForm.elements.namedItem("participant_no").value = String(
              userItem.participant_no || "",
            );
            userUpdateForm.elements.namedItem("nisn").value = String(
              userItem.nisn || "",
            );
            userUpdateForm.elements.namedItem("role").value = String(
              userItem.role || "siswa",
            );
            applyUserFormRoleRequirements(userUpdateForm);
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
      const createRoleSelect = userCreateForm.elements.namedItem("role");
      if (createRoleSelect) {
        createRoleSelect.addEventListener("change", function () {
          applyUserFormRoleRequirements(userCreateForm);
        });
      }
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
          const role = String(fd.get("role") || "")
            .trim()
            .toLowerCase();
          const participantNo = String(fd.get("participant_no") || "").trim();
          const nisn = String(fd.get("nisn") || "").trim();
          if (isStudentRole(role) && (schoolID === null || classID === null)) {
            setMsg("Untuk role siswa, sekolah dan kelas wajib dipilih.");
            return;
          }
          if (isStudentRole(role) && (!participantNo || !nisn)) {
            setMsg("Untuk role siswa, nomor peserta dan NISN wajib diisi.");
            return;
          }
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
            participant_no: participantNo,
            nisn: nisn,
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
      const updateRoleSelect = userUpdateForm.elements.namedItem("role");
      if (updateRoleSelect) {
        updateRoleSelect.addEventListener("change", function () {
          applyUserFormRoleRequirements(userUpdateForm);
        });
      }
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
          const role = String(fd.get("role") || "")
            .trim()
            .toLowerCase();
          const participantNo = String(fd.get("participant_no") || "").trim();
          const nisn = String(fd.get("nisn") || "").trim();
          if (isStudentRole(role) && (schoolID === null || classID === null)) {
            setMsg("Untuk role siswa, sekolah dan kelas wajib dipilih.");
            return;
          }
          if (isStudentRole(role) && (!participantNo || !nisn)) {
            setMsg("Untuk role siswa, nomor peserta dan NISN wajib diisi.");
            return;
          }
          if (schoolID !== null && classID === null) {
            setMsg("Jika memilih sekolah, kelas wajib dipilih.");
            return;
          }
          const out = await api("/api/v1/admin/users/" + id, "PUT", {
            full_name: String(fd.get("full_name") || "").trim(),
            email: String(fd.get("email") || "").trim(),
            role: String(fd.get("role") || "").trim(),
            password: String(fd.get("password") || ""),
            participant_no: participantNo,
            nisn: nisn,
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
        applyUserFormRoleRequirements(userCreateForm);
        fillSchoolSelect(userCreateForm.elements.namedItem("school_id"));
        resetClassSelect(
          userCreateForm.elements.namedItem("class_id"),
          "Pilih Sekolah Dulu",
        );
      }
      if (userUpdateForm) {
        applyUserFormRoleRequirements(userUpdateForm);
        fillSchoolSelect(userUpdateForm.elements.namedItem("school_id"));
        resetClassSelect(
          userUpdateForm.elements.namedItem("class_id"),
          "Pilih Sekolah Dulu",
        );
      }
      await loadUsers();
      await loadDashboardStats();
      await loadMasterSummaryTable();
      activateMenu("dashboard");
      setMsg("Dashboard admin siap.");
    } catch (err) {
      setMsg("Gagal memuat data awal admin: " + err.message);
    }
  }

  async function initProktorPage() {
    const root = document.querySelector('[data-page="proktor_content"]');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }
    if (!["admin", "proktor"].includes(String(user.role || ""))) {
      alert("Halaman proktor hanya untuk admin/proktor");
      window.location.href = "/";
      return;
    }

    const msg = document.getElementById("proktor-message");
    const statPending = document.getElementById("proktor-stat-pending");
    const statUsers = document.getElementById("proktor-stat-users");
    const statSchools = document.getElementById("proktor-stat-schools");
    const regRefreshBtn = document.getElementById(
      "proktor-registrations-refresh-btn",
    );
    const regBody = document.getElementById("proktor-registrations-body");
    const eventsForm = document.getElementById("proktor-events-form");
    const eventsBody = document.getElementById("proktor-events-body");
    const tokenPanel = document.getElementById("proktor-token-panel");
    const tokenForm = document.getElementById("proktor-token-form");
    const tokenExamSelect = document.getElementById("proktor-token-exam");
    const tokenTTLInput = document.getElementById("proktor-token-ttl");
    const tokenResult = document.getElementById("proktor-token-result");
    const examsPanel = document.getElementById("proktor-exams-panel");
    const examsRefreshBtn = document.getElementById(
      "proktor-exams-refresh-btn",
    );
    const examForm = document.getElementById("proktor-exam-form");
    const examSubjectSelect = document.getElementById("proktor-exam-subject");
    const examsBody = document.getElementById("proktor-exams-body");
    const examAssignForm = document.getElementById("proktor-exam-assign-form");
    const assignExamIDInput = document.getElementById("proktor-assign-exam-id");
    const assignSchoolSelect = document.getElementById("proktor-assign-school");
    const assignClassSelect = document.getElementById("proktor-assign-class");
    const examAssignmentsLoadBtn = document.getElementById(
      "proktor-exam-assignments-load-btn",
    );
    const examAssignmentsBody = document.getElementById(
      "proktor-exam-assignments-body",
    );
    const examQuestionForm = document.getElementById(
      "proktor-exam-question-form",
    );
    const questionExamIDInput = document.getElementById(
      "proktor-question-exam-id",
    );
    const examQuestionsLoadBtn = document.getElementById(
      "proktor-exam-questions-load-btn",
    );
    const examQuestionsBody = document.getElementById(
      "proktor-exam-questions-body",
    );
    const masterRefreshBtn = document.getElementById(
      "proktor-master-refresh-btn",
    );
    const levelsBody = document.getElementById("proktor-levels-body");
    const schoolsBody = document.getElementById("proktor-schools-body");
    const classesBody = document.getElementById("proktor-classes-body");
    const placementForm = document.getElementById("proktor-placement-form");
    const placementUserSelect = document.getElementById(
      "proktor-placement-user",
    );
    const placementSchoolSelect = document.getElementById(
      "proktor-placement-school",
    );
    const placementClassSelect = document.getElementById(
      "proktor-placement-class",
    );
    const registrationsPanel = document.getElementById(
      "proktor-registrations-panel",
    );
    const dashboardPanel = document.getElementById("proktor-dashboard-panel");
    const eventsPanel = document.getElementById("proktor-events-panel");
    const resultsPanel = document.getElementById("proktor-results-panel");
    const resultsRefreshBtn = document.getElementById(
      "proktor-results-refresh-btn",
    );
    const resultsAttemptForm = document.getElementById(
      "proktor-results-attempt-form",
    );
    const resultsSummary = document.getElementById("proktor-results-summary");
    const resultsExtra = document.getElementById("proktor-results-extra");
    const resultsReportBtn = document.getElementById(
      "proktor-results-report-btn",
    );
    const resultsStatsBtn = document.getElementById(
      "proktor-results-stats-btn",
    );
    const resultsChartBtn = document.getElementById(
      "proktor-results-chart-btn",
    );
    const masterPanel = document.getElementById("proktor-master-panel");
    const quickPanel = document.getElementById("proktor-quick-panel");
    const menuButtons = root.querySelectorAll("[data-proktor-menu]");
    let examFormSubmitting = false;
    let latestResultData = null;
    let placementUsers = [];
    let placementSchools = [];
    let placementClasses = [];

    const setMsg = function (value) {
      text(msg, value);
    };
    const localSchoolLabel = function (it) {
      const name = String((it && it.name) || "").trim();
      const code = String((it && it.code) || "").trim();
      if (!name) return "-";
      return code ? name + " (" + code + ")" : name;
    };
    const fmtDate = function (raw) {
      if (!raw) return "-";
      const d = new Date(raw);
      if (Number.isNaN(d.getTime())) return String(raw);
      return d.toLocaleString("id-ID");
    };

    const setTokenResult = function (value) {
      text(tokenResult, value);
      if (tokenResult) tokenResult.classList.remove("muted");
    };
    const setResultsSummary = function (value) {
      text(resultsSummary, value);
      if (resultsSummary) resultsSummary.classList.remove("muted");
    };
    const setResultsExtra = function (value) {
      text(resultsExtra, value);
      if (resultsExtra) resultsExtra.classList.remove("muted");
    };

    function summarizeAttemptResult(mode) {
      const data = latestResultData;
      const summary = data && data.summary ? data.summary : {};
      const items = Array.isArray(data && data.items) ? data.items : [];
      if (!data) {
        setResultsExtra("Muat nilai (Attempt ID) terlebih dahulu.");
        return;
      }
      const total = items.length;
      const correct = items.filter(function (it) {
        return !!it.is_correct;
      }).length;
      const wrong = items.filter(function (it) {
        return it.is_correct === false;
      }).length;
      const unanswered = items.filter(function (it) {
        return String(it.reason || "") === "unanswered";
      }).length;
      const score = Number(summary.score || 0);
      if (mode === "report") {
        setResultsExtra(
          "Laporan: exam #" +
            String(summary.exam_id || "-") +
            ", attempt #" +
            String(summary.id || "-") +
            ", skor " +
            String(score) +
            ", benar " +
            String(correct) +
            ", salah " +
            String(wrong) +
            ", kosong " +
            String(unanswered) +
            ".",
        );
        return;
      }
      if (mode === "stats") {
        const pct = total > 0 ? Math.round((correct / total) * 100) : 0;
        setResultsExtra(
          "Statistik: total soal " +
            String(total) +
            ", akurasi " +
            String(pct) +
            "%, benar " +
            String(correct) +
            ", salah " +
            String(wrong) +
            ", kosong " +
            String(unanswered) +
            ".",
        );
        return;
      }
      const bar = total > 0 ? Math.round((correct / total) * 20) : 0;
      setResultsExtra(
        "Grafik (teks): [" +
          "#".repeat(Math.max(0, bar)) +
          "-".repeat(Math.max(0, 20 - bar)) +
          "] " +
          String(correct) +
          "/" +
          String(total) +
          " benar.",
      );
    }
    function resetAssignClassSelect(placeholder) {
      if (!assignClassSelect) return;
      const textValue = String(placeholder || "Pilih kelas...");
      assignClassSelect.innerHTML =
        '<option value="">' + escapeHtml(textValue) + "</option>";
      assignClassSelect.disabled = true;
    }

    function fillAssignSchoolSelect(selectedSchoolID) {
      if (!assignSchoolSelect) return;
      assignSchoolSelect.innerHTML =
        '<option value="">Pilih sekolah...</option>';
      placementSchools.forEach(function (it) {
        const o = document.createElement("option");
        const id = Number(it.id || 0);
        o.value = String(id);
        o.textContent = String(it.name || "-");
        if (selectedSchoolID > 0 && id === selectedSchoolID) o.selected = true;
        assignSchoolSelect.appendChild(o);
      });
    }

    function fillAssignClassSelectBySchool(schoolID) {
      if (!assignClassSelect) return;
      const sid = Number(schoolID || 0);
      assignClassSelect.innerHTML = '<option value="">Pilih kelas...</option>';
      const items = placementClasses.filter(function (it) {
        return Number(it.school_id || 0) === sid;
      });
      if (!items.length) {
        resetAssignClassSelect("Tidak ada kelas di sekolah ini");
        return;
      }
      assignClassSelect.disabled = false;
      items.forEach(function (it) {
        const o = document.createElement("option");
        o.value = String(it.id || "");
        o.textContent =
          String(it.grade_level || "") + " | " + String(it.name || "-");
        assignClassSelect.appendChild(o);
      });
    }

    function resetPlacementClassSelect(placeholder) {
      if (!placementClassSelect) return;
      const textValue = String(placeholder || "Pilih kelas...");
      placementClassSelect.innerHTML =
        '<option value="">' + escapeHtml(textValue) + "</option>";
      placementClassSelect.disabled = true;
    }

    function fillPlacementSchoolSelect(selectedSchoolID) {
      if (!placementSchoolSelect) return;
      placementSchoolSelect.innerHTML =
        '<option value="">Pilih sekolah...</option>';
      placementSchools.forEach(function (it) {
        const o = document.createElement("option");
        const id = Number(it.id || 0);
        o.value = String(id);
        o.textContent = String(it.name || "-");
        if (selectedSchoolID > 0 && id === selectedSchoolID) o.selected = true;
        placementSchoolSelect.appendChild(o);
      });
    }

    function fillPlacementClassSelectBySchool(schoolID, selectedClassID) {
      if (!placementClassSelect) return;
      const sid = Number(schoolID || 0);
      placementClassSelect.innerHTML =
        '<option value="">Pilih kelas...</option>';
      const items = placementClasses.filter(function (it) {
        return Number(it.school_id || 0) === sid;
      });
      if (!items.length) {
        resetPlacementClassSelect("Tidak ada kelas di sekolah ini");
        return;
      }
      placementClassSelect.disabled = false;
      items.forEach(function (it) {
        const o = document.createElement("option");
        const classID = Number(it.id || 0);
        o.value = String(classID);
        o.textContent =
          String(it.grade_level || "") + " | " + String(it.name || "-");
        if (
          Number(selectedClassID || 0) > 0 &&
          classID === Number(selectedClassID || 0)
        ) {
          o.selected = true;
        }
        placementClassSelect.appendChild(o);
      });
    }

    async function loadPlacementOptions() {
      const [guru, siswa, schools, classes] = await Promise.all([
        api("/api/v1/admin/users?role=guru&limit=200&offset=0", "GET"),
        api("/api/v1/admin/users?role=siswa&limit=500&offset=0", "GET"),
        api("/api/v1/admin/schools?all=1", "GET"),
        api("/api/v1/admin/classes?all=1", "GET"),
      ]);
      placementUsers = []
        .concat(Array.isArray(guru.items) ? guru.items : [])
        .concat(Array.isArray(siswa.items) ? siswa.items : []);
      placementSchools = Array.isArray(schools) ? schools : [];
      placementClasses = Array.isArray(classes) ? classes : [];

      if (placementUserSelect) {
        placementUserSelect.innerHTML =
          '<option value="">Pilih pengguna...</option>';
        placementUsers.forEach(function (it) {
          const o = document.createElement("option");
          const id = Number(it.id || 0);
          const schoolID = Number(it.school_id || 0);
          const classID = Number(it.class_id || 0);
          o.value = String(id);
          o.setAttribute(
            "data-school-id",
            schoolID > 0 ? String(schoolID) : "",
          );
          o.setAttribute("data-class-id", classID > 0 ? String(classID) : "");
          o.textContent =
            String(it.full_name || it.username || "User") +
            " (" +
            String(it.role || "-") +
            ")" +
            (it.class_name ? " | " + String(it.class_name) : "");
          placementUserSelect.appendChild(o);
        });
      }
      fillPlacementSchoolSelect(0);
      resetPlacementClassSelect("Pilih sekolah dulu");
      fillAssignSchoolSelect(0);
      resetAssignClassSelect("Pilih sekolah dulu");
    }

    function activateProktorMenu(menu) {
      const allowed = [
        "dashboard",
        "registrations",
        "events",
        "token",
        "exams",
        "results",
        "master",
        "quick",
      ];
      const key = allowed.includes(menu) ? menu : "dashboard";
      menuButtons.forEach(function (btn) {
        const active = btn.getAttribute("data-proktor-menu") === key;
        btn.classList.toggle("active", active);
      });
      if (dashboardPanel) dashboardPanel.hidden = key !== "dashboard";
      if (registrationsPanel)
        registrationsPanel.hidden = key !== "registrations";
      if (eventsPanel) eventsPanel.hidden = key !== "events";
      if (tokenPanel) tokenPanel.hidden = key !== "token";
      if (examsPanel) examsPanel.hidden = key !== "exams";
      if (resultsPanel) resultsPanel.hidden = key !== "results";
      if (masterPanel) masterPanel.hidden = key !== "master";
      if (quickPanel) quickPanel.hidden = key !== "quick";
    }

    async function loadTokenExamOptions() {
      if (!tokenExamSelect) return;
      const exams = await api("/api/v1/admin/exams", "GET");
      tokenExamSelect.innerHTML = '<option value="">Pilih ujian...</option>';
      if (!Array.isArray(exams) || !exams.length) return;
      exams.forEach(function (it) {
        const o = document.createElement("option");
        o.value = String(it.id || "");
        const chunks = [
          String(it.code || "").trim(),
          String(it.title || "").trim(),
          String(it.subject_name || "").trim(),
        ].filter(Boolean);
        o.textContent = chunks.join(" | ");
        tokenExamSelect.appendChild(o);
      });
    }

    async function loadExamSubjectOptions() {
      if (!examSubjectSelect) return;
      const subjects = await api("/api/v1/subjects", "GET");
      examSubjectSelect.innerHTML = '<option value="">Pilih mapel...</option>';
      if (!Array.isArray(subjects)) return;
      subjects.forEach(function (it) {
        const o = document.createElement("option");
        o.value = String(it.id || "");
        o.textContent =
          String(it.education_level || "") +
          " | " +
          String(it.subject_type || "") +
          " | " +
          String(it.name || "");
        examSubjectSelect.appendChild(o);
      });
    }

    async function loadExamManageTable() {
      const exams = await api("/api/v1/admin/exams/manage", "GET");
      if (!examsBody) return;
      if (!Array.isArray(exams) || !exams.length) {
        examsBody.innerHTML =
          '<tr><td colspan="7" class="muted">Belum ada data ujian.</td></tr>';
        return;
      }
      examsBody.innerHTML = "";
      exams.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" +
            escapeHtml(String(it.code || "")) +
            "<br><small>" +
            escapeHtml(String(it.title || "")) +
            "</small></td>",
          "<td>" + escapeHtml(String(it.subject_name || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.duration_minutes || 0)) + " menit</td>",
          "<td>" + escapeHtml(String(it.question_count || 0)) + "</td>",
          "<td>" + escapeHtml(String(it.assigned_count || 0)) + "</td>",
          '<td><span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-exam-action="use" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Pilih Ujian" aria-label="Pilih Ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M5 12h14M13 6l6 6-6 6"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-exam-action="delete" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Nonaktifkan Ujian" aria-label="Nonaktifkan Ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/></svg>' +
            "</button>" +
            "</span></td>",
        ].join("");
        examsBody.appendChild(tr);
      });
    }

    async function loadExamQuestions(examID) {
      if (!examQuestionsBody) return;
      if (!examID || examID <= 0) {
        examQuestionsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Pilih exam lalu muat soal ujian.</td></tr>';
        return;
      }
      const items = await api(
        "/api/v1/admin/exams/" + Number(examID) + "/questions",
        "GET",
      );
      if (!Array.isArray(items) || !items.length) {
        examQuestionsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Belum ada soal di ujian ini.</td></tr>';
        return;
      }
      examQuestionsBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.seq_no || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_type || "")) + "</td>",
          "<td>" + escapeHtml(String(it.stem_preview || "")) + "</td>",
          "<td>" + escapeHtml(String(it.weight || 1)) + "</td>",
          '<td><button class="icon-only-btn danger" type="button" data-exam-question-del="' +
            escapeHtml(String(it.question_id || "")) +
            '" title="Hapus dari ujian" aria-label="Hapus dari ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/></svg>' +
            "</button></td>",
        ].join("");
        examQuestionsBody.appendChild(tr);
      });
    }

    async function loadExamAssignments(examID) {
      if (!examAssignmentsBody) return;
      if (!examID || examID <= 0) {
        examAssignmentsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Isi Exam ID lalu muat peserta ujian.</td></tr>';
        return;
      }
      const items = await api(
        "/api/v1/admin/exams/" + Number(examID) + "/assignments",
        "GET",
      );
      if (!Array.isArray(items) || !items.length) {
        examAssignmentsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Belum ada peserta terdaftar.</td></tr>';
        return;
      }
      examAssignmentsBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.user_id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.full_name || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.username || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.role || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.status || "-")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.assigned_at)) + "</td>",
        ].join("");
        examAssignmentsBody.appendChild(tr);
      });
    }

    async function loadPendingRegistrations() {
      const out = await api(
        "/api/v1/admin/registrations?status=pending&limit=200&offset=0",
        "GET",
      );
      const items = Array.isArray(out.items) ? out.items : [];
      if (statPending) statPending.textContent = String(items.length);
      if (!regBody) return items;
      if (!items.length) {
        regBody.innerHTML =
          '<tr><td colspan="6" class="muted">Tidak ada pendaftaran pending.</td></tr>';
        return items;
      }
      regBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.full_name || "")) + "</td>",
          "<td>" + escapeHtml(String(it.email || "")) + "</td>",
          "<td>" + escapeHtml(String(it.role_requested || "")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.created_at)) + "</td>",
          '<td><span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-action="approve" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Setujui" aria-label="Setujui">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M5 13l4 4L19 7"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-action="reject" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Tolak" aria-label="Tolak">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M18 6L6 18M6 6l12 12"/></svg>' +
            "</button>" +
            "</span></td>",
        ].join("");
        regBody.appendChild(tr);
      });
      return items;
    }

    async function loadProktorStatsAndPending() {
      const [stats] = await Promise.all([
        api("/api/v1/admin/dashboard/stats", "GET"),
        loadPendingRegistrations(),
      ]);
      const totalUsers =
        Number(stats.admin_count || 0) +
        Number(stats.proktor_count || 0) +
        Number(stats.guru_count || 0) +
        Number(stats.siswa_count || 0);
      if (statUsers) statUsers.textContent = String(totalUsers);
      if (statSchools)
        statSchools.textContent = String(stats.school_count || 0);
    }

    async function loadMasterReadonly() {
      const [levels, schools, classes] = await Promise.all([
        api("/api/v1/levels?all=1", "GET"),
        api("/api/v1/admin/schools?all=1", "GET"),
        api("/api/v1/admin/classes?all=1", "GET"),
      ]);
      const levelItems = Array.isArray(levels) ? levels : [];
      const schoolItems = Array.isArray(schools) ? schools : [];
      const classItems = Array.isArray(classes) ? classes : [];
      const schoolByID = {};
      schoolItems.forEach(function (it) {
        schoolByID[String(it.id || "")] = it;
      });

      if (levelsBody) {
        if (!levelItems.length) {
          levelsBody.innerHTML =
            '<tr><td colspan="3" class="muted">Tidak ada data jenjang.</td></tr>';
        } else {
          levelsBody.innerHTML = "";
          levelItems.forEach(function (it) {
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
              "<td>" + (it.is_active ? "aktif" : "nonaktif") + "</td>",
            ].join("");
            levelsBody.appendChild(tr);
          });
        }
      }

      if (schoolsBody) {
        if (!schoolItems.length) {
          schoolsBody.innerHTML =
            '<tr><td colspan="4" class="muted">Tidak ada data sekolah.</td></tr>';
        } else {
          schoolsBody.innerHTML = "";
          schoolItems.forEach(function (it) {
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
              "<td>" + escapeHtml(String(it.code || "-")) + "</td>",
              "<td>" + escapeHtml(String(it.address || "-")) + "</td>",
            ].join("");
            schoolsBody.appendChild(tr);
          });
        }
      }

      if (classesBody) {
        if (!classItems.length) {
          classesBody.innerHTML =
            '<tr><td colspan="4" class="muted">Tidak ada data kelas.</td></tr>';
        } else {
          classesBody.innerHTML = "";
          classItems.forEach(function (it) {
            const school = schoolByID[String(it.school_id || "")] || {};
            const schoolText = localSchoolLabel(school);
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(String(it.id || "")) + "</td>",
              "<td>" + escapeHtml(String(schoolText || "-")) + "</td>",
              "<td>" + escapeHtml(String(it.grade_level || "")) + "</td>",
              "<td>" + escapeHtml(String(it.name || "")) + "</td>",
            ].join("");
            classesBody.appendChild(tr);
          });
        }
      }
    }

    try {
      if (String((user && user.role) || "") === "proktor") {
        if (examQuestionForm) examQuestionForm.hidden = true;
        if (examQuestionsLoadBtn) examQuestionsLoadBtn.hidden = true;
      }
      await loadProktorStatsAndPending();
      await loadMasterReadonly();
      await loadPlacementOptions();
      await loadTokenExamOptions();
      await loadExamSubjectOptions();
      await loadExamManageTable();
      const firstMenu =
        (menuButtons &&
          menuButtons.length > 0 &&
          menuButtons[0].getAttribute("data-proktor-menu")) ||
        "dashboard";
      activateProktorMenu(firstMenu);
      setMsg("Dashboard proktor siap.");
    } catch (err) {
      setMsg("Gagal memuat dashboard proktor: " + err.message);
    }

    menuButtons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        activateProktorMenu(btn.getAttribute("data-proktor-menu") || "");
      });
    });
    root.querySelectorAll("[data-proktor-jump]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        activateProktorMenu(
          btn.getAttribute("data-proktor-jump") || "dashboard",
        );
      });
    });

    if (regRefreshBtn) {
      regRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          regRefreshBtn,
          msg,
          "Memuat pendaftaran pending...",
        );
        try {
          await loadPendingRegistrations();
          setMsg("Data pendaftaran pending berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat pendaftaran pending: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (regBody) {
      regBody.addEventListener("click", async function (e) {
        const btn =
          e.target && e.target.closest("button[data-action][data-id]");
        if (!btn) return;
        const action = String(btn.getAttribute("data-action") || "");
        const id = Number(btn.getAttribute("data-id") || 0);
        if (!id) return;

        const done = beginBusy(btn, msg, "Memproses pendaftaran...");
        try {
          if (action === "approve") {
            await api(
              "/api/v1/admin/registrations/" + id + "/approve",
              "POST",
              {},
            );
            setMsg("Pendaftaran berhasil disetujui.");
          } else if (action === "reject") {
            const note = window.prompt("Alasan penolakan (opsional):", "");
            if (note === null) return;
            await api("/api/v1/admin/registrations/" + id + "/reject", "POST", {
              note: String(note || "").trim(),
            });
            setMsg("Pendaftaran berhasil ditolak.");
          }
          await loadPendingRegistrations();
        } catch (err) {
          setMsg("Gagal memproses pendaftaran: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (eventsForm) {
      eventsForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(eventsForm);
        const attemptID = Number(fd.get("attempt_id") || 0);
        const limit = Number(fd.get("limit") || 100);
        if (attemptID <= 0) {
          setMsg("Attempt ID tidak valid.");
          return;
        }
        const done = beginBusy(eventsForm, msg, "Memuat event ujian...");
        try {
          const items = await api(
            "/api/v1/attempts/" +
              attemptID +
              "/events?limit=" +
              encodeURIComponent(String(limit)),
            "GET",
          );
          if (!eventsBody) return;
          if (!Array.isArray(items) || !items.length) {
            eventsBody.innerHTML =
              '<tr><td colspan="4" class="muted">Tidak ada event untuk attempt ini.</td></tr>';
            setMsg("Data event kosong.");
            return;
          }
          eventsBody.innerHTML = "";
          items.forEach(function (it) {
            const tr = document.createElement("tr");
            tr.innerHTML = [
              "<td>" + escapeHtml(fmtDate(it.server_ts)) + "</td>",
              "<td>" + escapeHtml(String(it.event_type || "")) + "</td>",
              "<td>" + escapeHtml(String(it.actor_user_id || "-")) + "</td>",
              "<td><code>" +
                escapeHtml(String(it.payload || "{}")) +
                "</code></td>",
            ].join("");
            eventsBody.appendChild(tr);
          });
          setMsg("Event ujian berhasil dimuat.");
        } catch (err) {
          if (eventsBody) {
            eventsBody.innerHTML =
              '<tr><td colspan="4" class="muted">Gagal memuat event.</td></tr>';
          }
          setMsg("Gagal memuat event: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (tokenForm) {
      tokenForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const examID = Number((tokenExamSelect && tokenExamSelect.value) || 0);
        const ttlMinutes = Number(
          (tokenTTLInput && tokenTTLInput.value) || 120,
        );
        if (examID <= 0) {
          setMsg("Pilih ujian terlebih dahulu.");
          return;
        }
        const done = beginBusy(tokenForm, msg, "Membuat token ujian...");
        try {
          const out = await api(
            "/api/v1/admin/exams/" + examID + "/token",
            "POST",
            { ttl_minutes: ttlMinutes },
          );
          const expiresAt =
            out && out.expires_at ? fmtDate(out.expires_at) : "-";
          setTokenResult(
            "Token: " +
              String((out && out.token) || "-") +
              " | Berlaku sampai: " +
              expiresAt,
          );
          setMsg("Token ujian berhasil dibuat.");
        } catch (err) {
          setMsg("Gagal membuat token ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examsRefreshBtn) {
      examsRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(examsRefreshBtn, msg, "Memuat data ujian...");
        try {
          await Promise.all([loadExamManageTable(), loadTokenExamOptions()]);
          setMsg("Data ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat data ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examForm) {
      examForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        if (examFormSubmitting) {
          setMsg("Penyimpanan ujian sedang diproses. Mohon tunggu...");
          return;
        }
        const fd = new FormData(examForm);
        const examID = Number(fd.get("exam_id") || 0);
        const payload = {
          code: String(fd.get("code") || "").trim(),
          title: String(fd.get("title") || "").trim(),
          subject_id: Number(fd.get("subject_id") || 0),
          duration_minutes: Number(fd.get("duration_minutes") || 90),
          review_policy: String(fd.get("review_policy") || "after_submit"),
          is_active: true,
        };
        examFormSubmitting = true;
        const done = beginBusy(examForm, msg, "Menyimpan ujian...");
        let saved = null;
        try {
          if (examID > 0) {
            saved = await api("/api/v1/admin/exams/" + examID, "PUT", payload);
          } else {
            saved = await api("/api/v1/admin/exams", "POST", payload);
          }

          const savedID = saved && saved.id ? String(saved.id) : "-";
          const savedCode = saved && saved.code ? String(saved.code) : "-";
          setMsg(
            "Ujian berhasil disimpan. ID: " + savedID + " | Kode: " + savedCode,
          );

          examForm.reset();
          const examIDInput = examForm.elements.namedItem("exam_id");
          if (examIDInput) examIDInput.value = "";
          try {
            await Promise.all([loadExamManageTable(), loadTokenExamOptions()]);
          } catch (refreshErr) {
            setMsg(
              "Ujian sudah tersimpan, tetapi gagal memuat ulang tabel: " +
                refreshErr.message,
            );
          }
        } catch (err) {
          setMsg("Gagal menyimpan ujian: " + err.message);
        } finally {
          done();
          window.setTimeout(function () {
            examFormSubmitting = false;
          }, 700);
        }
      });
    }

    if (examsBody) {
      examsBody.addEventListener("click", async function (e) {
        const useBtn =
          e.target && e.target.closest("button[data-exam-action][data-id]");
        if (!useBtn) return;
        const examID = Number(useBtn.getAttribute("data-id") || 0);
        const action = String(useBtn.getAttribute("data-exam-action") || "");
        if (examID <= 0) return;
        if (action === "use") {
          if (assignExamIDInput) assignExamIDInput.value = String(examID);
          if (questionExamIDInput) questionExamIDInput.value = String(examID);
          try {
            await Promise.all([
              loadExamQuestions(examID),
              loadExamAssignments(examID),
            ]);
            setMsg("Exam ID " + examID + " dipilih untuk enroll dan naskah.");
          } catch (err) {
            setMsg("Gagal memuat data ujian: " + err.message);
          }
          return;
        }
        if (action === "delete") {
          if (!window.confirm("Nonaktifkan ujian ini?")) return;
          const done = beginBusy(useBtn, msg, "Menonaktifkan ujian...");
          try {
            await api("/api/v1/admin/exams/" + examID, "DELETE");
            await Promise.all([loadExamManageTable(), loadTokenExamOptions()]);
            setMsg("Ujian berhasil dinonaktifkan.");
          } catch (err) {
            setMsg("Gagal menonaktifkan ujian: " + err.message);
          } finally {
            done();
          }
        }
      });
    }

    if (examAssignForm) {
      examAssignForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const examID = Number(
          (assignExamIDInput && assignExamIDInput.value) || 0,
        );
        if (examID <= 0) {
          setMsg("Isi Exam ID untuk enroll peserta.");
          return;
        }
        const schoolID = Number(
          (assignSchoolSelect && assignSchoolSelect.value) || 0,
        );
        const classID = Number(
          (assignClassSelect && assignClassSelect.value) || 0,
        );
        if (schoolID <= 0 || classID <= 0) {
          setMsg("Pilih sekolah dan kelas untuk enroll peserta.");
          return;
        }
        const done = beginBusy(
          examAssignForm,
          msg,
          "Menyimpan enroll peserta...",
        );
        try {
          const items = await api(
            "/api/v1/admin/exams/" + examID + "/assignments/by-class",
            "PUT",
            { school_id: schoolID, class_id: classID },
          );
          setMsg(
            "Enroll peserta berhasil disimpan. Total peserta aktif: " +
              String((Array.isArray(items) && items.length) || 0),
          );
          await Promise.all([
            loadExamManageTable(),
            loadExamAssignments(examID),
          ]);
        } catch (err) {
          setMsg("Gagal menyimpan enroll peserta: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examAssignmentsLoadBtn) {
      examAssignmentsLoadBtn.addEventListener("click", async function () {
        const examID = Number(
          (assignExamIDInput && assignExamIDInput.value) || 0,
        );
        const done = beginBusy(
          examAssignmentsLoadBtn,
          msg,
          "Memuat peserta ujian...",
        );
        try {
          await loadExamAssignments(examID);
          setMsg("Peserta ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat peserta ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (assignSchoolSelect) {
      assignSchoolSelect.addEventListener("change", function () {
        fillAssignClassSelectBySchool(assignSchoolSelect.value);
      });
    }

    if (examQuestionForm) {
      examQuestionForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(examQuestionForm);
        const examID = Number(fd.get("exam_id") || 0);
        if (examID <= 0) {
          setMsg("Isi Exam ID untuk menambah naskah.");
          return;
        }
        const payload = {
          question_id: Number(fd.get("question_id") || 0),
          seq_no: Number(fd.get("seq_no") || 0),
          weight: Number(fd.get("weight") || 1),
        };
        const done = beginBusy(
          examQuestionForm,
          msg,
          "Menyimpan naskah ke ujian...",
        );
        try {
          await api(
            "/api/v1/admin/exams/" + examID + "/questions",
            "POST",
            payload,
          );
          await Promise.all([loadExamQuestions(examID), loadExamManageTable()]);
          setMsg("Naskah berhasil dimasukkan ke ujian.");
        } catch (err) {
          setMsg("Gagal menyimpan naskah ke ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examQuestionsLoadBtn) {
      examQuestionsLoadBtn.addEventListener("click", async function () {
        const examID = Number(
          (questionExamIDInput && questionExamIDInput.value) || 0,
        );
        const done = beginBusy(
          examQuestionsLoadBtn,
          msg,
          "Memuat soal ujian...",
        );
        try {
          await loadExamQuestions(examID);
          setMsg("Soal ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat soal ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examQuestionsBody) {
      examQuestionsBody.addEventListener("click", async function (e) {
        const btn =
          e.target && e.target.closest("button[data-exam-question-del]");
        if (!btn) return;
        const examID = Number(
          (questionExamIDInput && questionExamIDInput.value) || 0,
        );
        const questionID = Number(
          btn.getAttribute("data-exam-question-del") || 0,
        );
        if (examID <= 0 || questionID <= 0) return;
        if (!window.confirm("Hapus soal ini dari ujian?")) return;
        const done = beginBusy(btn, msg, "Menghapus soal dari ujian...");
        try {
          await api(
            "/api/v1/admin/exams/" + examID + "/questions/" + questionID,
            "DELETE",
            {},
          );
          await Promise.all([loadExamQuestions(examID), loadExamManageTable()]);
          setMsg("Soal berhasil dihapus dari ujian.");
        } catch (err) {
          setMsg("Gagal menghapus soal dari ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (masterRefreshBtn) {
      masterRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          masterRefreshBtn,
          msg,
          "Memuat data master proktor...",
        );
        try {
          await loadMasterReadonly();
          await loadPlacementOptions();
          setMsg("Data master proktor berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat data master proktor: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (placementSchoolSelect) {
      placementSchoolSelect.addEventListener("change", function () {
        fillPlacementClassSelectBySchool(placementSchoolSelect.value, 0);
      });
    }

    if (placementUserSelect) {
      placementUserSelect.addEventListener("change", function () {
        const opt =
          placementUserSelect.options[placementUserSelect.selectedIndex] ||
          null;
        const schoolID = Number(
          (opt && opt.getAttribute("data-school-id")) || 0,
        );
        const classID = Number((opt && opt.getAttribute("data-class-id")) || 0);
        if (schoolID > 0) {
          fillPlacementSchoolSelect(schoolID);
          fillPlacementClassSelectBySchool(schoolID, classID);
          return;
        }
        fillPlacementSchoolSelect(0);
        resetPlacementClassSelect("Pilih sekolah dulu");
      });
    }

    if (placementForm) {
      placementForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const userID = Number(
          (placementUserSelect && placementUserSelect.value) || 0,
        );
        const schoolID = Number(
          (placementSchoolSelect && placementSchoolSelect.value) || 0,
        );
        const classID = Number(
          (placementClassSelect && placementClassSelect.value) || 0,
        );
        if (userID <= 0 || schoolID <= 0 || classID <= 0) {
          setMsg("Pilih pengguna, sekolah, dan kelas terlebih dahulu.");
          return;
        }
        const done = beginBusy(
          placementForm,
          msg,
          "Menyimpan penempatan kelas...",
        );
        try {
          const out = await api(
            "/api/v1/admin/users/" + userID + "/class-placement",
            "PUT",
            { school_id: schoolID, class_id: classID },
          );
          const userLabel = String(
            (out && out.full_name) || (out && out.username) || userID,
          );
          const classLabel = String((out && out.class_name) || classID);
          setMsg(
            "Penempatan berhasil: " + userLabel + " -> " + classLabel + ".",
          );
          await loadPlacementOptions();
        } catch (err) {
          setMsg("Gagal menyimpan penempatan kelas: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsRefreshBtn) {
      resultsRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          resultsRefreshBtn,
          msg,
          "Memuat daftar ujian...",
        );
        try {
          await loadExamManageTable();
          setMsg("Daftar ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat daftar ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsAttemptForm) {
      resultsAttemptForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(resultsAttemptForm);
        const attemptID = Number(fd.get("attempt_id") || 0);
        if (attemptID <= 0) {
          setMsg("Attempt ID tidak valid.");
          return;
        }
        const done = beginBusy(
          resultsAttemptForm,
          msg,
          "Memuat nilai attempt...",
        );
        try {
          const out = await api(
            "/api/v1/attempts/" + attemptID + "/result",
            "GET",
          );
          latestResultData = out || null;
          const summary = (out && out.summary) || {};
          const itemCount = Array.isArray(out && out.items)
            ? out.items.length
            : 0;
          setResultsSummary(
            "Exam #" +
              String(summary.exam_id || "-") +
              " | Attempt #" +
              String(summary.id || "-") +
              " | Skor: " +
              String(summary.score || 0) +
              " | Benar: " +
              String(summary.total_correct || 0) +
              " | Salah: " +
              String(summary.total_wrong || 0) +
              " | Kosong: " +
              String(summary.total_unanswered || 0) +
              " | Item: " +
              String(itemCount),
          );
          setResultsExtra("Nilai dimuat. Pilih Muat Laporan/Statistik/Grafik.");
          setMsg("Nilai attempt berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat nilai: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsReportBtn) {
      resultsReportBtn.addEventListener("click", function () {
        summarizeAttemptResult("report");
      });
    }
    if (resultsStatsBtn) {
      resultsStatsBtn.addEventListener("click", function () {
        summarizeAttemptResult("stats");
      });
    }
    if (resultsChartBtn) {
      resultsChartBtn.addEventListener("click", function () {
        summarizeAttemptResult("chart");
      });
    }
  }

  async function initGuruPage() {
    const root = document.querySelector('[data-page="guru_content"]');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = "/login";
      return;
    }
    if (!["admin", "guru"].includes(String(user.role || ""))) {
      alert("Halaman guru hanya untuk admin/guru");
      window.location.href = "/";
      return;
    }
    const canManageSubjects = String(user.role || "") === "guru";

    const msg = document.getElementById("guru-message");
    const statSubjects = document.getElementById("guru-stat-subjects");
    const statLevels = document.getElementById("guru-stat-levels");
    const statReviewPending = document.getElementById(
      "guru-stat-review-pending",
    );
    const statReviewTotal = document.getElementById("guru-stat-review-total");
    const reviewFilterForm = document.getElementById(
      "guru-reviews-filter-form",
    );
    const reviewRefreshBtn = document.getElementById(
      "guru-reviews-refresh-btn",
    );
    const reviewBody = document.getElementById("guru-reviews-body");
    const subjectRefreshBtn = document.getElementById(
      "guru-subjects-refresh-btn",
    );
    const subjectAddBtn = document.getElementById("guru-subjects-add-btn");
    const subjectsBody = document.getElementById("guru-subjects-body");
    const subjectDialog = document.getElementById("guru-subject-dialog");
    const subjectDialogForm = document.getElementById("guru-subject-form");
    const subjectLevelSelect = document.getElementById(
      "guru-subject-education-level",
    );
    const subjectDialogTitle = document.getElementById(
      "guru-subject-dialog-title",
    );
    const subjectCancelBtn = document.getElementById("guru-subject-cancel-btn");
    const subjectDeleteDialog = document.getElementById(
      "guru-subject-delete-dialog",
    );
    const subjectDeleteForm = document.getElementById(
      "guru-subject-delete-form",
    );
    const subjectDeleteLabel = document.getElementById(
      "guru-subject-delete-label",
    );
    const subjectDeleteCancelBtn = document.getElementById(
      "guru-subject-delete-cancel-btn",
    );
    const dashboardPanel = document.getElementById("guru-dashboard-panel");
    const reviewsPanel = document.getElementById("guru-reviews-panel");
    const blueprintsPanel = document.getElementById("guru-blueprints-panel");
    const stimuliPanel = document.getElementById("guru-stimuli-panel");
    const manuscriptsPanel = document.getElementById("guru-manuscripts-panel");
    const subjectsPanel = document.getElementById("guru-subjects-panel");
    const examsPanel = document.getElementById("guru-exams-panel");
    const resultsPanel = document.getElementById("guru-results-panel");
    const menuButtons = root.querySelectorAll("[data-guru-menu]");

    const blueprintForm = document.getElementById("guru-blueprint-form");
    const blueprintRefreshBtn = document.getElementById(
      "guru-blueprint-refresh-btn",
    );
    const blueprintFilterForm = document.getElementById(
      "guru-blueprint-filter-form",
    );
    const blueprintBody = document.getElementById("guru-blueprint-body");
    const blueprintSubjectSelect = document.getElementById(
      "guru-blueprint-subject-select",
    );
    const blueprintFilterSubject = document.getElementById(
      "guru-blueprint-filter-subject",
    );

    const stimulusForm = document.getElementById("guru-stimulus-form");
    const stimulusImportForm = document.getElementById(
      "guru-stimulus-import-form",
    );
    const stimulusTemplateBtn = document.getElementById(
      "guru-stimulus-template-btn",
    );
    const stimulusImportFileInput = document.getElementById(
      "guru-stimulus-import-file",
    );
    const stimulusListForm = document.getElementById("guru-stimulus-list-form");
    const stimulusRefreshBtn = document.getElementById(
      "guru-stimulus-refresh-btn",
    );
    const stimulusBody = document.getElementById("guru-stimulus-body");
    const stimulusInlineError = document.getElementById(
      "guru-stimulus-inline-error",
    );
    const stimulusLivePreview = document.getElementById(
      "guru-stimulus-live-preview",
    );
    const stimulusSubjectSelect = document.getElementById(
      "guru-stimulus-subject-select",
    );
    const stimulusListSubject = document.getElementById(
      "guru-stimulus-list-subject",
    );
    const stimulusSearchInput = document.getElementById("guru-stimulus-search");
    const stimulusPageSizeSelect = document.getElementById(
      "guru-stimulus-page-size",
    );
    const stimulusPrevBtn = document.getElementById("guru-stimulus-prev-btn");
    const stimulusNextBtn = document.getElementById("guru-stimulus-next-btn");
    const stimulusPageInfo = document.getElementById("guru-stimulus-page-info");
    const stimulusTypeSelect =
      stimulusForm && stimulusForm.elements.namedItem("stimulus_type");
    const stimulusSinglePanel = document.getElementById(
      "guru-stimulus-single-panel",
    );
    const stimulusSingleEditor = document.getElementById(
      "guru-stimulus-single-editor",
    );
    const stimulusMultiteksPanel = document.getElementById(
      "guru-stimulus-multiteks-panel",
    );
    const stimulusTabsContainer = document.getElementById("guru-stimulus-tabs");
    const stimulusAddTabBtn = document.getElementById(
      "guru-stimulus-add-tab-btn",
    );
    const stimulusPreviewDialog = document.getElementById(
      "guru-stimulus-preview-dialog",
    );
    const stimulusPreviewTitle = document.getElementById(
      "guru-stimulus-preview-title",
    );
    const stimulusPreviewContent = document.getElementById(
      "guru-stimulus-preview-content",
    );
    const stimulusEditDialog = document.getElementById(
      "guru-stimulus-edit-dialog",
    );
    const stimulusEditForm = document.getElementById("guru-stimulus-edit-form");
    const stimulusEditSubject = document.getElementById(
      "guru-stimulus-edit-subject",
    );
    const stimulusEditCancelBtn = document.getElementById(
      "guru-stimulus-edit-cancel-btn",
    );
    const stimulusDeleteDialog = document.getElementById(
      "guru-stimulus-delete-dialog",
    );
    const stimulusDeleteForm = document.getElementById(
      "guru-stimulus-delete-form",
    );
    const stimulusDeleteLabel = document.getElementById(
      "guru-stimulus-delete-label",
    );
    const stimulusDeleteCancelBtn = document.getElementById(
      "guru-stimulus-delete-cancel-btn",
    );
    const stimulusImportErrorDialog = document.getElementById(
      "guru-stimulus-import-error-dialog",
    );
    const stimulusImportErrorOutput = document.getElementById(
      "guru-stimulus-import-error-output",
    );
    const stimulusImportErrorCloseBtn = document.getElementById(
      "guru-stimulus-import-error-close-btn",
    );
    const stimulusImportDownloadBtn = document.getElementById(
      "guru-stimulus-import-download-btn",
    );

    const manuscriptForm = document.getElementById("guru-manuscript-form");
    const manuscriptImportBtn = document.getElementById(
      "guru-manuscript-import-btn",
    );
    const questionOwnerScopeSelect = document.getElementById(
      "guru-question-owner-scope",
    );
    const manuscriptImportDialog = document.getElementById(
      "guru-manuscript-import-dialog",
    );
    const manuscriptImportForm = document.getElementById(
      "guru-manuscript-import-form",
    );
    const manuscriptImportSubjectSelect = document.getElementById(
      "guru-manuscript-import-subject-select",
    );
    const manuscriptImportFileInput = document.getElementById(
      "guru-manuscript-import-file",
    );
    const manuscriptImportOutput = document.getElementById(
      "guru-manuscript-import-output",
    );
    const manuscriptImportCancelBtn = document.getElementById(
      "guru-manuscript-import-cancel-btn",
    );
    const manuscriptRefreshBtn = document.getElementById(
      "guru-manuscript-refresh-btn",
    );
    const manuscriptListForm = document.getElementById(
      "guru-manuscript-list-form",
    );
    const manuscriptFinalizeForm = document.getElementById(
      "guru-manuscript-finalize-form",
    );
    const manuscriptBody = document.getElementById("guru-manuscript-body");
    const manuscriptPageSizeSelect = document.getElementById(
      "guru-manuscript-page-size",
    );
    const manuscriptPrevBtn = document.getElementById(
      "guru-manuscript-prev-btn",
    );
    const manuscriptNextBtn = document.getElementById(
      "guru-manuscript-next-btn",
    );
    const manuscriptPageInfo = document.getElementById(
      "guru-manuscript-page-info",
    );
    const manuscriptMessage = document.getElementById(
      "guru-manuscript-message",
    );
    const manuscriptQuestionSelect = document.getElementById(
      "guru-manuscript-question-select",
    );
    const manuscriptQuestionTypeUI = document.getElementById(
      "guru-manuscript-question-type-ui",
    );
    const manuscriptStimulusSelect = document.getElementById(
      "guru-manuscript-stimulus-select",
    );
    const manuscriptListQuestionSelect = document.getElementById(
      "guru-manuscript-list-question-select",
    );
    const manuscriptFinalizeQuestionSelect = document.getElementById(
      "guru-manuscript-finalize-question-select",
    );
    const manuscriptStemEditor = document.getElementById(
      "guru-manuscript-stem-editor",
    );
    const manuscriptStemHTMLInput = document.getElementById(
      "guru-manuscript-stem-html",
    );
    const manuscriptExplanationEditor = document.getElementById(
      "guru-manuscript-explanation-editor",
    );
    const manuscriptExplanationHTMLInput = document.getElementById(
      "guru-manuscript-explanation-html",
    );
    const manuscriptHintEditor = document.getElementById(
      "guru-manuscript-hint-editor",
    );
    const manuscriptHintHTMLInput = document.getElementById(
      "guru-manuscript-hint-html",
    );
    const manuscriptOptionsPanel = document.getElementById(
      "guru-manuscript-options-panel",
    );
    const manuscriptOptionsWrap = document.getElementById(
      "guru-manuscript-options",
    );
    const manuscriptAddOptionBtn = document.getElementById(
      "guru-manuscript-add-option-btn",
    );
    const manuscriptKeyMC = document.getElementById("guru-manuscript-key-mc");
    const manuscriptKeyMR = document.getElementById("guru-manuscript-key-mr");
    const manuscriptKeyMCOptions = document.getElementById(
      "guru-manuscript-key-mc-options",
    );
    const manuscriptKeyMROptions = document.getElementById(
      "guru-manuscript-key-mr-options",
    );
    const manuscriptKeyTF = document.getElementById("guru-manuscript-key-tf");
    const manuscriptStatementsWrap = document.getElementById(
      "guru-manuscript-statements",
    );
    const manuscriptAddStatementBtn = document.getElementById(
      "guru-manuscript-add-statement-btn",
    );
    const manuscriptAnswerKeyRaw = document.getElementById(
      "guru-manuscript-answer-key-raw",
    );
    const manuscriptEditingVersionNoInput = document.getElementById(
      "guru-manuscript-editing-version-no",
    );
    const manuscriptSubmitBtn = document.getElementById(
      "guru-manuscript-submit-btn",
    );
    const manuscriptCancelEditBtn = document.getElementById(
      "guru-manuscript-cancel-edit-btn",
    );
    const manuscriptPreviewDialog = document.getElementById(
      "guru-manuscript-preview-dialog",
    );
    const manuscriptPreviewTitle = document.getElementById(
      "guru-manuscript-preview-title",
    );
    const manuscriptPreviewContent = document.getElementById(
      "guru-manuscript-preview-content",
    );
    const examsRefreshBtn = document.getElementById("guru-exams-refresh-btn");
    const examForm = document.getElementById("guru-exam-form");
    const examSubjectSelect = document.getElementById("guru-exam-subject");
    const examsBody = document.getElementById("guru-exams-body");
    const examQuestionForm = document.getElementById("guru-exam-question-form");
    const questionExamIDInput = document.getElementById(
      "guru-question-exam-id",
    );
    const examQuestionsLoadBtn = document.getElementById(
      "guru-exam-questions-load-btn",
    );
    const examQuestionsBody = document.getElementById(
      "guru-exam-questions-body",
    );
    const resultsRefreshBtn = document.getElementById(
      "guru-results-refresh-btn",
    );
    const resultsAssignmentForm = document.getElementById(
      "guru-results-assignment-form",
    );
    const resultsExamIDInput = document.getElementById("guru-results-exam-id");
    const resultsAssignmentBody = document.getElementById(
      "guru-results-assignment-body",
    );
    const resultsAttemptForm = document.getElementById(
      "guru-results-attempt-form",
    );
    const resultsSummary = document.getElementById("guru-results-summary");
    const resultsExtra = document.getElementById("guru-results-extra");
    const resultsReportBtn = document.getElementById("guru-results-report-btn");
    const resultsStatsBtn = document.getElementById("guru-results-stats-btn");
    const resultsChartBtn = document.getElementById("guru-results-chart-btn");

    const reviewDecisionDialog = document.getElementById(
      "guru-review-decision-dialog",
    );
    const reviewDecisionForm = document.getElementById(
      "guru-review-decision-form",
    );
    const reviewDecisionTitle = document.getElementById(
      "guru-review-decision-title",
    );
    const reviewDecisionHelp = document.getElementById(
      "guru-review-decision-help",
    );
    const reviewDecisionNote = document.getElementById("guru-review-note");
    const reviewDecisionCancelBtn = document.getElementById(
      "guru-review-decision-cancel-btn",
    );
    let subjectsCache = [];
    let blueprintsCache = [];
    const stimuliBySubject = {};
    let currentStimulusTableSubjectID = 0;
    let stimulusSingleQuill = null;
    const tabQuillByWrap = new WeakMap();
    let manuscriptStemQuill = null;
    let manuscriptExplanationQuill = null;
    let manuscriptHintQuill = null;
    const manuscriptOptionQuillByRow = new WeakMap();
    let currentManuscriptQuestionType = "";
    let manuscriptVersionsCache = [];
    let manuscriptAllItems = [];
    let manuscriptPage = 1;
    let manuscriptPageSize = 15;
    let manuscriptEditingQuestionID = 0;
    let currentStimulusItems = [];
    let educationLevelsCache = [];
    let currentStimulusAllItems = [];
    let stimulusListPage = 1;
    let stimulusListPageSize = 10;
    let stimulusListSearch = "";
    let lastStimulusImportErrors = [];
    let examFormSubmitting = false;
    let latestResultData = null;

    const setMsg = function (value) {
      text(msg, value);
    };
    const setManuscriptMsg = function (value) {
      const message = String(value || "");
      if (manuscriptMessage) text(manuscriptMessage, message);
      setMsg(message);
    };
    const setResultsSummary = function (value) {
      text(resultsSummary, value);
      if (resultsSummary) resultsSummary.classList.remove("muted");
    };
    const setResultsExtra = function (value) {
      text(resultsExtra, value);
      if (resultsExtra) resultsExtra.classList.remove("muted");
    };

    function summarizeAttemptResult(mode) {
      const data = latestResultData;
      const summary = data && data.summary ? data.summary : {};
      const items = Array.isArray(data && data.items) ? data.items : [];
      if (!data) {
        setResultsExtra("Muat nilai (Attempt ID) terlebih dahulu.");
        return;
      }
      const total = items.length;
      const correct = items.filter(function (it) {
        return !!it.is_correct;
      }).length;
      const wrong = items.filter(function (it) {
        return it.is_correct === false;
      }).length;
      const unanswered = items.filter(function (it) {
        return String(it.reason || "") === "unanswered";
      }).length;
      const score = Number(summary.score || 0);
      if (mode === "report") {
        setResultsExtra(
          "Laporan: exam #" +
            String(summary.exam_id || "-") +
            ", attempt #" +
            String(summary.id || "-") +
            ", skor " +
            String(score) +
            ", benar " +
            String(correct) +
            ", salah " +
            String(wrong) +
            ", kosong " +
            String(unanswered) +
            ".",
        );
        return;
      }
      if (mode === "stats") {
        const pct = total > 0 ? Math.round((correct / total) * 100) : 0;
        setResultsExtra(
          "Statistik: total soal " +
            String(total) +
            ", akurasi " +
            String(pct) +
            "%, benar " +
            String(correct) +
            ", salah " +
            String(wrong) +
            ", kosong " +
            String(unanswered) +
            ".",
        );
        return;
      }
      const bar = total > 0 ? Math.round((correct / total) * 20) : 0;
      setResultsExtra(
        "Grafik (teks): [" +
          "#".repeat(Math.max(0, bar)) +
          "-".repeat(Math.max(0, 20 - bar)) +
          "] " +
          String(correct) +
          "/" +
          String(total) +
          " benar.",
      );
    }

    function isOwnerOnlyQuestionsEnabled() {
      const scope = String(
        (questionOwnerScopeSelect && questionOwnerScopeSelect.value) || "",
      )
        .trim()
        .toLowerCase();
      return scope === "mine";
    }

    async function loadEducationLevelsForGuruSubjectForm() {
      try {
        const out = await api("/api/v1/levels", "GET");
        educationLevelsCache = Array.isArray(out) ? out : [];
      } catch (_) {
        educationLevelsCache = [];
      }
    }

    function fillSubjectEducationLevelSelect(selectedValue) {
      if (!subjectLevelSelect) return;
      const selected = String(selectedValue || "").trim();
      subjectLevelSelect.innerHTML = '<option value="">Pilih Jenjang</option>';
      educationLevelsCache.forEach(function (it) {
        const name = String((it && it.name) || "").trim();
        if (!name) return;
        const opt = document.createElement("option");
        opt.value = name;
        opt.textContent = name;
        if (selected && selected === name) opt.selected = true;
        subjectLevelSelect.appendChild(opt);
      });
      if (
        selected &&
        !Array.from(subjectLevelSelect.options).some(function (o) {
          return o.value === selected;
        })
      ) {
        const legacy = document.createElement("option");
        legacy.value = selected;
        legacy.textContent = selected + " (legacy)";
        legacy.selected = true;
        subjectLevelSelect.appendChild(legacy);
      }
    }

    const fmtDate = function (raw) {
      if (!raw) return "-";
      const d = new Date(raw);
      if (Number.isNaN(d.getTime())) return String(raw);
      return d.toLocaleString("id-ID");
    };

    function parseDownloadFilenameLite(disposition, fallback) {
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

    async function downloadStimulusTemplateCSV() {
      const res = await fetch("/api/v1/stimuli/import-template", {
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
      const filename = parseDownloadFilenameLite(
        res.headers.get("Content-Disposition"),
        "template_import_stimuli.csv",
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

    function formatStimulusImportErrors(errors) {
      const rows = Array.isArray(errors) ? errors : [];
      if (!rows.length) return "";
      return rows
        .slice(0, 10)
        .map(function (it) {
          const rowNo = Number(it && it.row) || 0;
          const title = String((it && it.title) || "").trim();
          const err = String((it && it.error) || "").trim();
          if (title) {
            return "Baris " + String(rowNo) + " (" + title + "): " + err;
          }
          return "Baris " + String(rowNo) + ": " + err;
        })
        .join(" | ");
    }

    function formatStimulusImportErrorsMultiline(errors) {
      const rows = Array.isArray(errors) ? errors : [];
      if (!rows.length) return "Tidak ada detail error.";
      return rows
        .map(function (it) {
          const rowNo = Number(it && it.row) || 0;
          const title = String((it && it.title) || "").trim();
          const err = String(
            (it && it.error) || "error tidak diketahui",
          ).trim();
          if (title) {
            return (
              "Baris " + String(rowNo) + " | judul: " + title + " | " + err
            );
          }
          return "Baris " + String(rowNo) + " | " + err;
        })
        .join("\n");
    }

    function csvEscape(value) {
      const s = String(value == null ? "" : value);
      if (!/[",\n]/.test(s)) return s;
      return '"' + s.replaceAll('"', '""') + '"';
    }

    function downloadStimulusImportErrorsCSV(errors) {
      const rows = Array.isArray(errors) ? errors : [];
      const lines = ["row,title,error"];
      rows.forEach(function (it) {
        lines.push(
          [
            csvEscape(Number(it && it.row) || 0),
            csvEscape(String((it && it.title) || "").trim()),
            csvEscape(String((it && it.error) || "").trim()),
          ].join(","),
        );
      });
      const now = new Date();
      const pad = function (n) {
        return String(n).padStart(2, "0");
      };
      const stamp =
        String(now.getFullYear()) +
        pad(now.getMonth() + 1) +
        pad(now.getDate()) +
        "_" +
        pad(now.getHours()) +
        pad(now.getMinutes()) +
        pad(now.getSeconds());
      const blob = new Blob([lines.join("\n") + "\n"], {
        type: "text/csv;charset=utf-8",
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "laporan_gagal_import_stimuli_" + stamp + ".csv";
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    }

    function decodeImportWordCSVLine(rawLine) {
      const line = String(rawLine == null ? "" : rawLine).trim();
      if (!line) return "";
      if (line.startsWith('"') && line.endsWith('"') && line.length >= 2) {
        return line.slice(1, -1).replaceAll('""', '"').trim();
      }
      return line.trim();
    }

    function parseImportWordBlocks(rawText) {
      const lines = String(rawText || "")
        .split(/\r?\n/)
        .map(function (line) {
          return decodeImportWordCSVLine(line);
        });
      const blocks = [];
      let current = [];
      lines.forEach(function (line, idx) {
        const clean = String(line || "").trim();
        if (!clean) return;
        if (idx === 0 && clean.toLowerCase() === "content") return;
        if (/^type\s*:/i.test(clean) && current.length) {
          blocks.push(current);
          current = [];
        }
        current.push(clean);
      });
      if (current.length) blocks.push(current);
      return blocks;
    }

    function mapImportQuestionType(rawType) {
      const t = String(rawType || "")
        .trim()
        .toUpperCase();
      if (t === "MC") return "pg_tunggal";
      if (t === "MR") return "multi_jawaban";
      if (t === "TF") return "benar_salah_pernyataan";
      return "";
    }

    function normalizeTextKey(raw) {
      return String(raw || "")
        .replace(/<[^>]*>/g, " ")
        .replace(/&nbsp;/gi, " ")
        .replace(/&amp;/gi, "&")
        .replace(/&lt;/gi, "<")
        .replace(/&gt;/gi, ">")
        .replace(/&#39;/gi, "'")
        .replace(/&quot;/gi, '"')
        .replace(/\s+/g, " ")
        .trim()
        .toLowerCase();
    }

    function buildAutoStimulusTitle(blockNo, qType) {
      const now = new Date();
      const pad = function (n) {
        return String(n).padStart(2, "0");
      };
      const stamp =
        String(now.getFullYear()) +
        pad(now.getMonth() + 1) +
        pad(now.getDate()) +
        "_" +
        pad(now.getHours()) +
        pad(now.getMinutes());
      const t = String(qType || "")
        .trim()
        .toUpperCase();
      return (
        "Stimulus Import " +
        stamp +
        " [" +
        (t || "SOAL") +
        "] #" +
        String(blockNo)
      ).slice(0, 200);
    }

    function readStimulusBodyText(stimulusItem) {
      const content = stimulusItem && stimulusItem.content;
      if (!content || typeof content !== "object") return "";
      return String(content.body || "").trim();
    }

    function parseImportQuestionBlock(lines, blockNo) {
      const item = {
        type_raw: "",
        question_type: "",
        stimulus: "",
        question: "",
        options: [],
        statements: [],
        key_raw: "",
      };
      (Array.isArray(lines) ? lines : []).forEach(function (line) {
        const m = String(line || "").match(/^([A-Za-z0-9_]+)\s*:\s*(.*)$/);
        if (!m) return;
        const key = String(m[1] || "")
          .trim()
          .toUpperCase();
        const val = String(m[2] || "").trim();
        if (!key) return;
        if (key === "TYPE") {
          item.type_raw = val;
          item.question_type = mapImportQuestionType(val);
          return;
        }
        if (key === "S" || key === "STIMULUS") {
          item.stimulus = val;
          return;
        }
        if (key === "Q") {
          item.question = val;
          return;
        }
        if (key === "KUNCI") {
          item.key_raw = val;
          return;
        }
        if (/^[A-Z]$/.test(key)) {
          item.options.push({ option_key: key, option_html: val });
          return;
        }
        if (/^P[0-9]+$/.test(key)) {
          item.statements.push({ id: key, statement: val });
        }
      });

      if (!item.question_type) {
        throw new Error("Blok " + String(blockNo) + ": TYPE harus MC/MR/TF.");
      }
      if (!item.question) {
        throw new Error("Blok " + String(blockNo) + ": Q wajib diisi.");
      }
      if (!item.key_raw) {
        throw new Error("Blok " + String(blockNo) + ": Kunci wajib diisi.");
      }

      let answerKey = {};
      let optionsPayload = [];
      if (item.question_type === "pg_tunggal") {
        optionsPayload = item.options
          .map(function (it) {
            return {
              option_key: String((it && it.option_key) || "")
                .trim()
                .toUpperCase(),
              option_html: String((it && it.option_html) || "").trim(),
            };
          })
          .filter(function (it) {
            return it.option_key && it.option_html;
          });
        if (optionsPayload.length < 2) {
          throw new Error(
            "Blok " + String(blockNo) + ": soal MC minimal 2 opsi.",
          );
        }
        const correct = String(item.key_raw || "")
          .split(",")[0]
          .trim()
          .toUpperCase();
        if (!correct) {
          throw new Error(
            "Blok " + String(blockNo) + ": kunci MC tidak valid.",
          );
        }
        answerKey = { correct: correct };
      } else if (item.question_type === "multi_jawaban") {
        optionsPayload = item.options
          .map(function (it) {
            return {
              option_key: String((it && it.option_key) || "")
                .trim()
                .toUpperCase(),
              option_html: String((it && it.option_html) || "").trim(),
            };
          })
          .filter(function (it) {
            return it.option_key && it.option_html;
          });
        if (optionsPayload.length < 2) {
          throw new Error(
            "Blok " + String(blockNo) + ": soal MR minimal 2 opsi.",
          );
        }
        const keys = String(item.key_raw || "")
          .split(",")
          .map(function (v) {
            return String(v || "")
              .trim()
              .toUpperCase();
          })
          .filter(Boolean);
        if (!keys.length) {
          throw new Error(
            "Blok " + String(blockNo) + ": kunci MR tidak valid.",
          );
        }
        answerKey = { correct: keys, mode: "exact" };
      } else {
        const statements = item.statements
          .map(function (it) {
            return {
              id: String((it && it.id) || "")
                .trim()
                .toUpperCase(),
              statement: String((it && it.statement) || "").trim(),
            };
          })
          .filter(function (it) {
            return it.id && it.statement;
          });
        if (!statements.length) {
          throw new Error(
            "Blok " + String(blockNo) + ": soal TF butuh P1/P2/P3...",
          );
        }
        const keys = String(item.key_raw || "")
          .split(",")
          .map(function (v) {
            return String(v || "")
              .trim()
              .toUpperCase();
          })
          .filter(Boolean);
        if (keys.length !== statements.length) {
          throw new Error(
            "Blok " +
              String(blockNo) +
              ": jumlah Kunci TF harus sama dengan jumlah pernyataan.",
          );
        }
        answerKey = {
          statements: statements.map(function (st, idx) {
            const flag = keys[idx];
            if (flag !== "T" && flag !== "F") {
              throw new Error(
                "Blok " +
                  String(blockNo) +
                  ": nilai Kunci TF harus T/F (contoh: T, F, T).",
              );
            }
            return {
              id: st.id,
              statement: st.statement,
              correct: flag === "T",
            };
          }),
        };
      }

      const safeQuestion = escapeHtml(item.question || "");
      const stemHTML = "<p>" + safeQuestion + "</p>";
      const title = String(item.question || "").slice(0, 180);
      return {
        question_type: item.question_type,
        title: title || "Import Soal",
        indicator: "Import Soal CSV Word",
        material: item.stimulus || "Import Soal",
        objective: "Import otomatis dari CSV Word",
        cognitive_level: "C2",
        stimulus_text: String(item.stimulus || "").trim(),
        stem_html: stemHTML,
        answer_key: answerKey,
        options: optionsPayload,
      };
    }

    async function loadScriptOnce(id, src) {
      const exists = document.getElementById(id);
      if (exists) return;
      await new Promise(function (resolve, reject) {
        const s = document.createElement("script");
        s.id = id;
        s.src = src;
        s.async = true;
        s.onload = resolve;
        s.onerror = function () {
          reject(new Error("Gagal memuat script: " + src));
        };
        document.head.appendChild(s);
      });
    }

    function loadStyleOnce(id, href) {
      if (document.getElementById(id)) return;
      const l = document.createElement("link");
      l.id = id;
      l.rel = "stylesheet";
      l.href = href;
      document.head.appendChild(l);
    }

    async function ensureQuillReady() {
      if (window.Quill) return;
      loadStyleOnce(
        "quill-snow-css",
        "https://cdn.jsdelivr.net/npm/quill@1.3.7/dist/quill.snow.css",
      );
      loadStyleOnce(
        "katex-css",
        "https://cdn.jsdelivr.net/npm/katex@0.16.10/dist/katex.min.css",
      );
      await loadScriptOnce(
        "katex-js",
        "https://cdn.jsdelivr.net/npm/katex@0.16.10/dist/katex.min.js",
      );
      await loadScriptOnce(
        "quill-js",
        "https://cdn.jsdelivr.net/npm/quill@1.3.7/dist/quill.min.js",
      );
    }

    function createQuillEditor(hostEl, placeholder) {
      if (!hostEl || !window.Quill) return null;
      return new window.Quill(hostEl, {
        theme: "snow",
        placeholder: placeholder || "",
        modules: {
          toolbar: {
            container: [
              [{ header: [false, 2, 3] }],
              ["bold", "italic", "underline"],
              [{ align: [] }],
              [{ list: "ordered" }, { list: "bullet" }],
              ["link", "image", "formula"],
              ["clean"],
            ],
            handlers: {
              image: function () {
                const input = document.createElement("input");
                input.type = "file";
                input.accept = "image/*";
                input.onchange = function () {
                  const file = input.files && input.files[0];
                  if (!file) return;
                  const reader = new FileReader();
                  reader.onload = function () {
                    const base64 = String(reader.result || "");
                    if (!base64) return;
                    const range = this.quill.getSelection(true);
                    const index = range ? range.index : this.quill.getLength();
                    this.quill.insertEmbed(index, "image", base64, "user");
                    this.quill.setSelection(index + 1, 0, "silent");
                  }.bind(this);
                  reader.readAsDataURL(file);
                }.bind(this);
                input.click();
              },
            },
          },
        },
      });
    }

    function buildStimulusTabEditor() {
      if (!stimulusTabsContainer) return null;
      const wrap = document.createElement("section");
      wrap.className = "panel mt stimulus-tab-item";
      wrap.innerHTML = [
        '<label>Judul Tab <input type="text" class="stimulus-tab-title" placeholder="Judul tab" required /></label>',
        '<div class="quill-editor-shell mt"><div class="stimulus-tab-editor"></div></div>',
        '<div class="mt"><button type="button" class="btn btn-secondary btn-inline" data-editor-action="remove-tab">Hapus Tab</button></div>',
      ].join("");
      stimulusTabsContainer.appendChild(wrap);
      const host = wrap.querySelector(".stimulus-tab-editor");
      const quill = createQuillEditor(host, "Isi konten tab...");
      if (quill) {
        tabQuillByWrap.set(wrap, quill);
        quill.on("text-change", function () {
          renderLiveStimulusPreview();
          clearStimulusInlineError();
        });
      }
      return wrap;
    }

    function currentStimulusType() {
      if (!stimulusTypeSelect) return "single";
      const v = String(stimulusTypeSelect.value || "")
        .trim()
        .toLowerCase();
      return v === "multiteks" ? "multiteks" : "single";
    }

    function switchStimulusEditorByType() {
      const t = currentStimulusType();
      if (stimulusSinglePanel) stimulusSinglePanel.hidden = t !== "single";
      if (stimulusMultiteksPanel)
        stimulusMultiteksPanel.hidden = t !== "multiteks";
      if (t === "multiteks" && stimulusTabsContainer) {
        if (!stimulusTabsContainer.querySelector(".stimulus-tab-item")) {
          buildStimulusTabEditor();
        }
      }
    }

    function getQuillHTML(quill) {
      if (!quill || !quill.root) return "";
      return String(quill.root.innerHTML || "").trim();
    }

    function syncManuscriptHiddenInputs() {
      if (manuscriptStemHTMLInput) {
        manuscriptStemHTMLInput.value = getQuillHTML(manuscriptStemQuill);
      }
      if (manuscriptExplanationHTMLInput) {
        manuscriptExplanationHTMLInput.value = getQuillHTML(
          manuscriptExplanationQuill,
        );
      }
      if (manuscriptHintHTMLInput) {
        manuscriptHintHTMLInput.value = getQuillHTML(manuscriptHintQuill);
      }
    }

    function resetManuscriptEditors() {
      if (manuscriptStemQuill) manuscriptStemQuill.setContents([]);
      if (manuscriptExplanationQuill)
        manuscriptExplanationQuill.setContents([]);
      if (manuscriptHintQuill) manuscriptHintQuill.setContents([]);
      syncManuscriptHiddenInputs();
    }

    function createManuscriptOptionRow(data, idx) {
      const key =
        String((data && data.option_key) || "").trim() ||
        String.fromCharCode(65 + idx);
      const htmlValue = String((data && data.option_html) || "").trim();
      return (
        '<div class="panel mt" data-option-row="1">' +
        '<label>Kode Opsi<input type="text" data-option-key value="' +
        escapeHtml(key) +
        '" placeholder="A" maxlength="10" /></label>' +
        '<label class="mt">Konten Opsi</label>' +
        '<div class="quill-editor-shell mt"><div class="manuscript-option-editor"></div></div>' +
        '<input type="hidden" data-option-html value="' +
        escapeHtml(htmlValue) +
        '" />' +
        '<button type="button" class="btn btn-secondary btn-inline mt" data-option-remove="1">Hapus Opsi</button>' +
        "</div>"
      );
    }

    function bindManuscriptOptionEditor(row) {
      if (!row || manuscriptOptionQuillByRow.has(row) || !window.Quill) return;
      const host = row.querySelector(".manuscript-option-editor");
      const hidden = row.querySelector("[data-option-html]");
      if (!host || !hidden) return;
      const initialHTML = String(hidden.value || "").trim();
      const quill = createQuillEditor(host, "Isi opsi jawaban...");
      if (!quill) return;
      if (initialHTML) {
        quill.clipboard.dangerouslyPasteHTML(initialHTML);
      }
      hidden.value = getQuillHTML(quill);
      quill.on("text-change", function () {
        hidden.value = getQuillHTML(quill);
        syncManuscriptAnswerKeyPreview();
      });
      manuscriptOptionQuillByRow.set(row, quill);
    }

    function bindManuscriptOptionEditors() {
      if (!manuscriptOptionsWrap) return;
      manuscriptOptionsWrap
        .querySelectorAll("[data-option-row]")
        .forEach(function (row) {
          bindManuscriptOptionEditor(row);
        });
    }

    function ensureManuscriptOptionRows() {
      if (!manuscriptOptionsWrap) return;
      if (!manuscriptOptionsWrap.querySelector("[data-option-row]")) {
        manuscriptOptionsWrap.innerHTML =
          createManuscriptOptionRow({ option_key: "A", option_html: "" }, 0) +
          createManuscriptOptionRow({ option_key: "B", option_html: "" }, 1);
      }
      bindManuscriptOptionEditors();
    }

    function collectManuscriptOptions() {
      const rows = Array.from(
        (manuscriptOptionsWrap &&
          manuscriptOptionsWrap.querySelectorAll("[data-option-row]")) ||
          [],
      );
      return rows.map(function (row) {
        const keyInput = row.querySelector("[data-option-key]");
        const htmlInput = row.querySelector("[data-option-html]");
        return {
          option_key: String((keyInput && keyInput.value) || "").trim(),
          option_html: String((htmlInput && htmlInput.value) || "").trim(),
        };
      });
    }

    function validateManuscriptOptions(options) {
      const list = Array.isArray(options) ? options : [];
      const isChoiceType =
        currentManuscriptQuestionType === "pg_tunggal" ||
        currentManuscriptQuestionType === "multi_jawaban";
      if (!isChoiceType) return [];
      if (list.length < 2) {
        throw new Error("Minimal 2 opsi jawaban untuk MC/MR.");
      }
      const seen = {};
      list.forEach(function (it, idx) {
        const key = String((it && it.option_key) || "")
          .trim()
          .toUpperCase();
        const body = String((it && it.option_html) || "").trim();
        if (!key)
          throw new Error(
            "Kode opsi baris " + String(idx + 1) + " wajib diisi.",
          );
        if (seen[key]) throw new Error("Kode opsi duplikat: " + key);
        if (!body) throw new Error("Konten opsi " + key + " wajib diisi.");
        seen[key] = true;
      });
      return list.map(function (it) {
        return {
          option_key: String(it.option_key || "")
            .trim()
            .toUpperCase(),
          option_html: String(it.option_html || "").trim(),
        };
      });
    }

    function renderManuscriptKeyChoices() {
      if (!manuscriptKeyMCOptions || !manuscriptKeyMROptions) return;
      const options = collectManuscriptOptions();
      const mcChecked = document.querySelector(
        'input[name="manuscript_mc_key"]:checked',
      );
      const mrChecked = Array.from(
        document.querySelectorAll('input[name="manuscript_mr_key"]:checked'),
      ).map(function (el) {
        return String(el.value || "")
          .trim()
          .toUpperCase();
      });
      const mcKey = String((mcChecked && mcChecked.value) || "")
        .trim()
        .toUpperCase();
      manuscriptKeyMCOptions.innerHTML = "";
      manuscriptKeyMROptions.innerHTML = "";
      options.forEach(function (it, idx) {
        const key = String((it && it.option_key) || "")
          .trim()
          .toUpperCase();
        const label = key || "Opsi " + String(idx + 1);
        const mcRow = document.createElement("label");
        mcRow.className = "option-row";
        mcRow.innerHTML =
          '<input type="radio" name="manuscript_mc_key" value="' +
          escapeHtml(key) +
          '" ' +
          ((mcKey ? mcKey === key : idx === 0) ? "checked" : "") +
          ">" +
          "<div>" +
          escapeHtml(label) +
          "</div>";
        manuscriptKeyMCOptions.appendChild(mcRow);

        const mrRow = document.createElement("label");
        mrRow.className = "option-row";
        mrRow.innerHTML =
          '<input type="checkbox" name="manuscript_mr_key" value="' +
          escapeHtml(key) +
          '" ' +
          (mrChecked.includes(key) ? "checked" : "") +
          ">" +
          "<div>" +
          escapeHtml(label) +
          "</div>";
        manuscriptKeyMROptions.appendChild(mrRow);
      });
    }

    function createTFStatementRow(data, idx) {
      const id = String((data && data.id) || "S" + String(idx + 1)).trim();
      const statement = String((data && data.statement) || "").trim();
      const correct = !!(data && data.correct);
      return (
        '<div class="panel mt" data-statement-row="1">' +
        '<label>ID Pernyataan<input type="text" data-statement-id value="' +
        escapeHtml(id) +
        '" placeholder="S1" /></label>' +
        '<label class="mt">Teks Pernyataan<input type="text" data-statement-text value="' +
        escapeHtml(statement) +
        '" placeholder="Teks pernyataan" /></label>' +
        '<label class="mt">Nilai Benar/Salah<select data-statement-correct><option value="true"' +
        (correct ? " selected" : "") +
        '>Benar</option><option value="false"' +
        (!correct ? " selected" : "") +
        ">Salah</option></select></label>" +
        '<button type="button" class="btn btn-secondary btn-inline mt" data-statement-remove="1">Hapus</button>' +
        "</div>"
      );
    }

    function ensureTFStatementRows() {
      if (!manuscriptStatementsWrap) return;
      if (!manuscriptStatementsWrap.querySelector("[data-statement-row]")) {
        manuscriptStatementsWrap.innerHTML = createTFStatementRow(
          { id: "S1", statement: "", correct: true },
          0,
        );
      }
    }

    function getCurrentManuscriptQuestionType() {
      const questionID = Number(
        (manuscriptQuestionSelect && manuscriptQuestionSelect.value) || 0,
      );
      const item = blueprintsCache.find(function (it) {
        return Number(it.id) === questionID;
      });
      return String((item && item.question_type) || "")
        .trim()
        .toLowerCase();
    }

    function collectManuscriptAnswerKey() {
      const qType = String(currentManuscriptQuestionType || "").toLowerCase();
      if (qType === "pg_tunggal") {
        renderManuscriptKeyChoices();
        const checked = document.querySelector(
          'input[name="manuscript_mc_key"]:checked',
        );
        const correct = String((checked && checked.value) || "").trim();
        if (!correct) throw new Error("Kunci MC wajib dipilih.");
        return { correct: correct };
      }
      if (qType === "multi_jawaban") {
        renderManuscriptKeyChoices();
        const keys = Array.from(
          document.querySelectorAll('input[name="manuscript_mr_key"]:checked'),
        )
          .map(function (el) {
            return String(el.value || "").trim();
          })
          .filter(Boolean);
        if (!keys.length) throw new Error("Kunci MR minimal 1 opsi.");
        return { correct: keys, mode: "exact" };
      }
      if (qType === "benar_salah_pernyataan") {
        const rows = Array.from(
          (manuscriptStatementsWrap &&
            manuscriptStatementsWrap.querySelectorAll(
              "[data-statement-row]",
            )) ||
            [],
        );
        if (!rows.length) {
          throw new Error("Tambahkan minimal 1 pernyataan TF.");
        }
        const seen = {};
        const statements = rows.map(function (row, idx) {
          const idInput = row.querySelector("[data-statement-id]");
          const textInput = row.querySelector("[data-statement-text]");
          const correctSelect = row.querySelector("[data-statement-correct]");
          const id = String((idInput && idInput.value) || "").trim();
          const statement = String((textInput && textInput.value) || "").trim();
          const correct = String(
            (correctSelect && correctSelect.value) || "true",
          );
          if (!id) {
            throw new Error(
              "ID pernyataan TF baris " + String(idx + 1) + " wajib diisi.",
            );
          }
          if (seen[id]) {
            throw new Error("ID pernyataan TF duplikat: " + id);
          }
          seen[id] = true;
          return {
            id: id,
            statement: statement,
            correct: correct === "true",
          };
        });
        return { statements: statements };
      }
      throw new Error("Tipe soal pada kisi-kisi belum valid.");
    }

    function syncManuscriptAnswerKeyPreview() {
      if (!manuscriptAnswerKeyRaw) return;
      try {
        const key = collectManuscriptAnswerKey();
        manuscriptAnswerKeyRaw.value = JSON.stringify(key, null, 2);
      } catch (_) {
        manuscriptAnswerKeyRaw.value = "{}";
      }
    }

    function applyManuscriptTypeUI(fromBlueprint) {
      const blueprintType = getCurrentManuscriptQuestionType();
      let selectedType = String(
        (manuscriptQuestionTypeUI && manuscriptQuestionTypeUI.value) || "",
      )
        .trim()
        .toLowerCase();
      if (fromBlueprint || !selectedType) {
        selectedType = blueprintType;
        if (manuscriptQuestionTypeUI) {
          manuscriptQuestionTypeUI.value = selectedType || "";
        }
      }
      currentManuscriptQuestionType = selectedType || blueprintType;
      if (manuscriptOptionsPanel) {
        manuscriptOptionsPanel.hidden = ![
          "pg_tunggal",
          "multi_jawaban",
        ].includes(currentManuscriptQuestionType);
      }
      if (manuscriptKeyMC) {
        manuscriptKeyMC.hidden = currentManuscriptQuestionType !== "pg_tunggal";
      }
      if (manuscriptKeyMR) {
        manuscriptKeyMR.hidden =
          currentManuscriptQuestionType !== "multi_jawaban";
      }
      if (manuscriptKeyTF) {
        manuscriptKeyTF.hidden =
          currentManuscriptQuestionType !== "benar_salah_pernyataan";
      }
      if (currentManuscriptQuestionType === "benar_salah_pernyataan") {
        ensureTFStatementRows();
      } else {
        ensureManuscriptOptionRows();
        renderManuscriptKeyChoices();
      }
      syncManuscriptAnswerKeyPreview();
    }

    function setManuscriptEditMode(questionID, versionNo) {
      manuscriptEditingQuestionID = Number(questionID || 0);
      if (manuscriptEditingVersionNoInput) {
        manuscriptEditingVersionNoInput.value = String(versionNo || "");
      }
      if (manuscriptSubmitBtn) {
        manuscriptSubmitBtn.textContent =
          "Simpan Perubahan Draft (v" + String(versionNo || "") + ")";
      }
      if (manuscriptCancelEditBtn) {
        manuscriptCancelEditBtn.hidden = false;
      }
    }

    function clearManuscriptEditMode() {
      manuscriptEditingQuestionID = 0;
      if (manuscriptEditingVersionNoInput)
        manuscriptEditingVersionNoInput.value = "";
      if (manuscriptSubmitBtn)
        manuscriptSubmitBtn.textContent = "Simpan Draft Naskah";
      if (manuscriptCancelEditBtn) manuscriptCancelEditBtn.hidden = true;
    }

    function setQuillHTMLValue(quill, htmlValue) {
      if (!quill) return;
      quill.setContents([]);
      const value = String(htmlValue || "").trim();
      if (value) quill.clipboard.dangerouslyPasteHTML(value);
    }

    function parseAnswerKeyObject(raw) {
      if (!raw) return {};
      if (typeof raw === "object") return raw;
      try {
        return JSON.parse(String(raw || "{}"));
      } catch (_) {
        return {};
      }
    }

    function renderManuscriptOptionsFromData(options) {
      if (!manuscriptOptionsWrap) return;
      const list = Array.isArray(options) ? options : [];
      if (!list.length) {
        ensureManuscriptOptionRows();
        return;
      }
      manuscriptOptionsWrap.innerHTML = list
        .map(function (it, idx) {
          return createManuscriptOptionRow(it || {}, idx);
        })
        .join("");
      bindManuscriptOptionEditors();
    }

    function renderTFStatementsFromData(statements) {
      if (!manuscriptStatementsWrap) return;
      const list = Array.isArray(statements) ? statements : [];
      if (!list.length) {
        ensureTFStatementRows();
        return;
      }
      manuscriptStatementsWrap.innerHTML = list
        .map(function (it, idx) {
          return createTFStatementRow(it || {}, idx);
        })
        .join("");
    }

    function applyAnswerKeyToManuscriptUI(answerKeyObj) {
      const qType = String(currentManuscriptQuestionType || "").toLowerCase();
      const key =
        answerKeyObj && typeof answerKeyObj === "object" ? answerKeyObj : {};
      if (qType === "pg_tunggal") {
        renderManuscriptKeyChoices();
        const correct = String(key.correct || "")
          .trim()
          .toUpperCase();
        const target = manuscriptKeyMCOptions
          ? manuscriptKeyMCOptions.querySelector(
              'input[name="manuscript_mc_key"][value="' + correct + '"]',
            )
          : null;
        if (target) target.checked = true;
      } else if (qType === "multi_jawaban") {
        renderManuscriptKeyChoices();
        const correctKeys = Array.isArray(key.correct)
          ? key.correct
              .map(function (it) {
                return String(it || "")
                  .trim()
                  .toUpperCase();
              })
              .filter(Boolean)
          : [];
        if (manuscriptKeyMROptions) {
          manuscriptKeyMROptions
            .querySelectorAll('input[name="manuscript_mr_key"]')
            .forEach(function (el) {
              el.checked = correctKeys.includes(
                String(el.value || "")
                  .trim()
                  .toUpperCase(),
              );
            });
        }
      } else if (qType === "benar_salah_pernyataan") {
        renderTFStatementsFromData(
          Array.isArray(key.statements) ? key.statements : [],
        );
      }
      syncManuscriptAnswerKeyPreview();
    }

    async function beginManuscriptEdit(item) {
      if (!item || !manuscriptForm) return;
      const questionID = Number(item.question_id || 0);
      const versionNo = Number(item.version_no || 0);
      if (questionID <= 0 || versionNo <= 0) return;

      if (manuscriptQuestionSelect) {
        manuscriptQuestionSelect.value = String(questionID);
      }
      applyManuscriptTypeUI(true);
      await refreshStimuliSelectByQuestion();

      if (manuscriptStimulusSelect) {
        manuscriptStimulusSelect.value = String(Number(item.stimulus_id || 0));
      }
      setQuillHTMLValue(manuscriptStemQuill, item.stem_html);
      setQuillHTMLValue(manuscriptExplanationQuill, item.explanation_html);
      setQuillHTMLValue(manuscriptHintQuill, item.hint_html);
      syncManuscriptHiddenInputs();

      const keyObj = parseAnswerKeyObject(item.answer_key);
      if (currentManuscriptQuestionType === "benar_salah_pernyataan") {
        renderTFStatementsFromData(keyObj.statements);
      } else {
        renderManuscriptOptionsFromData(item.options);
      }
      applyManuscriptTypeUI(false);
      applyAnswerKeyToManuscriptUI(keyObj);

      setManuscriptEditMode(questionID, versionNo);
      manuscriptForm.scrollIntoView({ behavior: "smooth", block: "start" });
    }

    function previewManuscriptVersion(item) {
      if (!item || !manuscriptPreviewContent) return;
      const keyObj = parseAnswerKeyObject(item.answer_key);
      const safeJSON = escapeHtml(JSON.stringify(keyObj, null, 2));
      const stem = String(item.stem_html || "").trim();
      const explanation = String(item.explanation_html || "").trim();
      const hint = String(item.hint_html || "").trim();
      const options = Array.isArray(item.options) ? item.options : [];
      const optionsHTML = options.length
        ? options
            .map(function (it) {
              return (
                '<div class="mt"><strong>' +
                escapeHtml(String((it && it.option_key) || "")) +
                "</strong><div>" +
                String((it && it.option_html) || "") +
                "</div></div>"
              );
            })
            .join("")
        : '<div class="muted">Tidak ada opsi tersimpan.</div>';
      if (manuscriptPreviewTitle) {
        text(
          manuscriptPreviewTitle,
          "Detail Naskah Q" +
            String(item.question_id || "-") +
            " v" +
            String(item.version_no || "-"),
        );
      }
      manuscriptPreviewContent.innerHTML = [
        "<div><strong>Status:</strong> " +
          escapeHtml(String(item.status || "")) +
          "</div>",
        '<div class="mt"><strong>Stem:</strong><div class="box mt">' +
          (stem || '<span class="muted">(kosong)</span>') +
          "</div></div>",
        '<div class="mt"><strong>Penjelasan:</strong><div class="box mt">' +
          (explanation || '<span class="muted">(kosong)</span>') +
          "</div></div>",
        '<div class="mt"><strong>Hint:</strong><div class="box mt">' +
          (hint || '<span class="muted">(kosong)</span>') +
          "</div></div>",
        '<div class="mt"><strong>Opsi:</strong><div class="box mt">' +
          optionsHTML +
          "</div></div>",
        '<div class="mt"><strong>Answer Key:</strong><pre class="code-box">' +
          safeJSON +
          "</pre></div>",
      ].join("");
      if (
        manuscriptPreviewDialog &&
        typeof manuscriptPreviewDialog.showModal === "function"
      ) {
        manuscriptPreviewDialog.showModal();
      }
    }

    function isBlankHTML(value) {
      const raw = String(value || "").trim();
      if (!raw) return true;
      const plain = raw
        .replace(/<[^>]*>/g, " ")
        .replace(/&nbsp;/gi, " ")
        .replace(/\s+/g, " ")
        .trim();
      return !plain;
    }

    function clearStimulusInlineError() {
      if (!stimulusInlineError) return;
      stimulusInlineError.hidden = true;
      stimulusInlineError.textContent = "";
    }

    function showStimulusInlineError(errors) {
      if (!stimulusInlineError) return;
      const rows = Array.isArray(errors) ? errors : [String(errors || "")];
      stimulusInlineError.textContent = rows.filter(Boolean).join(" | ");
      stimulusInlineError.hidden = rows.length === 0;
    }

    function collectStimulusContentPayload() {
      const t = currentStimulusType();
      if (t === "single") {
        return { body: getQuillHTML(stimulusSingleQuill) };
      }
      const tabs = [];
      if (stimulusTabsContainer) {
        stimulusTabsContainer
          .querySelectorAll(".stimulus-tab-item")
          .forEach(function (row) {
            const title = String(
              (row.querySelector(".stimulus-tab-title") || {}).value || "",
            ).trim();
            const quill = tabQuillByWrap.get(row);
            const body = getQuillHTML(quill);
            if (title || body) tabs.push({ title: title, body: body });
          });
      }
      return { tabs: tabs };
    }

    function validateStimulusFormPayload(fd, content) {
      const errors = [];
      const subjectID = Number(fd.get("subject_id") || 0);
      const title = String(fd.get("title") || "").trim();
      const stimulusType = String(fd.get("stimulus_type") || "").trim();
      if (subjectID <= 0) errors.push("Mapel wajib dipilih.");
      if (!title) errors.push("Judul stimulus wajib diisi.");
      if (stimulusType === "single" && !String(content.body || "").trim()) {
        errors.push("Konten single wajib diisi.");
      }
      if (stimulusType === "multiteks") {
        const tabs = Array.isArray(content.tabs) ? content.tabs : [];
        if (!tabs.length) errors.push("Tambahkan minimal 1 tab multiteks.");
        const hasIncomplete = tabs.some(function (tab) {
          const title = String((tab && tab.title) || "").trim();
          const body = String((tab && tab.body) || "").trim();
          return (title && !body) || (!title && body);
        });
        if (hasIncomplete) {
          errors.push("Setiap tab multiteks harus berisi Judul dan Konten.");
        }
      }
      return errors;
    }

    function renderLiveStimulusPreview() {
      if (!stimulusLivePreview || !stimulusForm) return;
      const fd = new FormData(stimulusForm);
      const title = String(fd.get("title") || "").trim();
      const stimulusType = String(fd.get("stimulus_type") || "single").trim();
      const content = collectStimulusContentPayload();
      let html = '<div class="muted">Konten preview kosong.</div>';
      if (stimulusType === "single") {
        const body = String(content.body || "").trim();
        if (body) html = body;
      } else {
        const tabs = Array.isArray(content.tabs) ? content.tabs : [];
        if (tabs.length) {
          html = tabs
            .map(function (tab, idx) {
              const tabTitle = String((tab && tab.title) || "").trim();
              const tabBody = String((tab && tab.body) || "").trim();
              return (
                '<section class="panel mt"><h4>' +
                escapeHtml(tabTitle || "Tab " + String(idx + 1)) +
                '</h4><div class="box mt">' +
                (tabBody || '<span class="muted">(konten kosong)</span>') +
                "</div></section>"
              );
            })
            .join("");
        }
      }
      stimulusLivePreview.innerHTML =
        "<strong>" +
        escapeHtml(title || "Tanpa Judul") +
        "</strong>" +
        '<div class="mt">' +
        html +
        "</div>";
    }

    function resetStimulusEditors() {
      if (stimulusSingleQuill) stimulusSingleQuill.setContents([]);
      if (stimulusTabsContainer) stimulusTabsContainer.innerHTML = "";
      if (currentStimulusType() === "multiteks") buildStimulusTabEditor();
    }

    function activateGuruMenu(menu) {
      const allowed = [
        "dashboard",
        "reviews",
        "blueprints",
        "stimuli",
        "manuscripts",
        "exams",
        "results",
        "subjects",
      ];
      const key = allowed.includes(menu) ? menu : "dashboard";
      menuButtons.forEach(function (btn) {
        const active = btn.getAttribute("data-guru-menu") === key;
        btn.classList.toggle("active", active);
      });
      if (dashboardPanel) dashboardPanel.hidden = key !== "dashboard";
      if (reviewsPanel) reviewsPanel.hidden = key !== "reviews";
      if (blueprintsPanel) blueprintsPanel.hidden = key !== "blueprints";
      if (stimuliPanel) stimuliPanel.hidden = key !== "stimuli";
      if (manuscriptsPanel) manuscriptsPanel.hidden = key !== "manuscripts";
      if (examsPanel) examsPanel.hidden = key !== "exams";
      if (resultsPanel) resultsPanel.hidden = key !== "results";
      if (subjectsPanel) subjectsPanel.hidden = key !== "subjects";
    }

    function subjectLabel(it) {
      const level = String((it && it.education_level) || "").trim();
      const stype = String((it && it.subject_type) || "").trim();
      const name = String((it && it.name) || "").trim();
      return [level, stype, name].filter(Boolean).join(" | ");
    }

    function fillSubjectSelect(selectEl, withAllOption) {
      if (!selectEl) return;
      const current = String(selectEl.value || "");
      selectEl.innerHTML = "";
      if (withAllOption) {
        const allOpt = document.createElement("option");
        allOpt.value = "";
        allOpt.textContent = "Semua Mapel";
        selectEl.appendChild(allOpt);
      }
      subjectsCache.forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id || "");
        opt.textContent = subjectLabel(it);
        if (current && current === String(it.id || "")) opt.selected = true;
        selectEl.appendChild(opt);
      });
      if (!selectEl.value && selectEl.options.length > 0) {
        selectEl.selectedIndex = 0;
      }
    }

    function blueprintLabel(it) {
      const title = String((it && it.title) || "").trim();
      const subject = String((it && it.subject_name) || "").trim();
      if (!title && !subject) return String((it && it.id) || "");
      if (!subject) return title;
      if (!title) return "#" + String((it && it.id) || "") + " - " + subject;
      return (
        "#" + String((it && it.id) || "") + " - " + title + " (" + subject + ")"
      );
    }

    function fillBlueprintSelect(selectEl) {
      if (!selectEl) return;
      const current = String(selectEl.value || "");
      selectEl.innerHTML = "";
      const firstOpt = document.createElement("option");
      firstOpt.value = "";
      firstOpt.textContent = "Pilih Kisi-Kisi";
      selectEl.appendChild(firstOpt);
      blueprintsCache.forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id || "");
        opt.textContent = blueprintLabel(it);
        if (current && current === String(it.id || "")) opt.selected = true;
        selectEl.appendChild(opt);
      });
      if (
        (selectEl === manuscriptQuestionSelect ||
          selectEl === manuscriptListQuestionSelect ||
          selectEl === manuscriptFinalizeQuestionSelect) &&
        !selectEl.value &&
        selectEl.options.length > 1
      ) {
        selectEl.selectedIndex = 1;
      }
    }

    async function loadAllManuscriptVersionsFromBlueprints() {
      const qids = Array.from(
        new Set(
          (Array.isArray(blueprintsCache) ? blueprintsCache : [])
            .map(function (it) {
              return Number((it && it.id) || 0);
            })
            .filter(function (v) {
              return v > 0;
            }),
        ),
      );
      if (!qids.length) return [];
      const settled = await Promise.allSettled(
        qids.map(function (qid) {
          return api("/api/v1/questions/" + qid + "/versions", "GET");
        }),
      );
      return settled
        .map(function (it) {
          return it && it.status === "fulfilled" ? it.value : [];
        })
        .reduce(function (acc, rows) {
          if (Array.isArray(rows)) return acc.concat(rows);
          return acc;
        }, [])
        .sort(function (a, b) {
          const ta = new Date(a && a.created_at ? a.created_at : 0).getTime();
          const tb = new Date(b && b.created_at ? b.created_at : 0).getTime();
          if (tb !== ta) return tb - ta;
          const qa = Number((a && a.question_id) || 0);
          const qb = Number((b && b.question_id) || 0);
          if (qb !== qa) return qb - qa;
          return (
            Number((b && b.version_no) || 0) - Number((a && a.version_no) || 0)
          );
        });
    }

    function fillStimulusSelect(selectEl, items) {
      if (!selectEl) return;
      const current = String(selectEl.value || "");
      selectEl.innerHTML = "";
      const firstOpt = document.createElement("option");
      firstOpt.value = "";
      firstOpt.textContent = "Pilih Stimulus";
      selectEl.appendChild(firstOpt);
      (items || []).forEach(function (it) {
        const opt = document.createElement("option");
        opt.value = String(it.id || "");
        opt.textContent =
          "#" + String(it.id || "") + " - " + String(it.title || "");
        if (current && current === String(it.id || "")) opt.selected = true;
        selectEl.appendChild(opt);
      });
    }

    function openSubjectDialog(editItem) {
      if (!subjectDialogForm || !subjectDialog) return;
      subjectDialogForm.reset();
      const idEl = subjectDialogForm.elements.namedItem("id");
      const typeEl = subjectDialogForm.elements.namedItem("subject_type");
      const nameEl = subjectDialogForm.elements.namedItem("name");
      if (idEl) idEl.value = editItem ? String(editItem.id || "") : "";
      fillSubjectEducationLevelSelect(
        editItem ? String(editItem.education_level || "") : "",
      );
      if (typeEl)
        typeEl.value = editItem ? String(editItem.subject_type || "") : "";
      if (nameEl) nameEl.value = editItem ? String(editItem.name || "") : "";
      if (subjectDialogTitle) {
        subjectDialogTitle.textContent = editItem
          ? "Ubah Mapel"
          : "Tambah Mapel";
      }
      if (typeof subjectDialog.showModal === "function") {
        subjectDialog.showModal();
      }
    }

    function openSubjectDeleteDialog(item) {
      if (!subjectDeleteDialog || !subjectDeleteForm) return;
      const idEl = subjectDeleteForm.elements.namedItem("id");
      if (idEl) idEl.value = String((item && item.id) || "");
      if (subjectDeleteLabel) {
        text(
          subjectDeleteLabel,
          "Mapel '" + String((item && item.name) || "") + "' akan dihapus.",
        );
      }
      if (typeof subjectDeleteDialog.showModal === "function") {
        subjectDeleteDialog.showModal();
      }
    }

    function renderBlueprints(items) {
      if (!blueprintBody) return;
      if (!Array.isArray(items) || items.length === 0) {
        blueprintBody.innerHTML =
          '<tr><td colspan="8" class="muted">Tidak ada data kisi-kisi.</td></tr>';
        return;
      }
      blueprintBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.subject_name || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_type || "")) + "</td>",
          "<td>" + escapeHtml(String(it.title || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.indicator || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.material || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.cognitive_level || "-")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.created_at)) + "</td>",
        ].join("");
        blueprintBody.appendChild(tr);
      });
    }

    function getFilteredStimulusItems() {
      const q = String(stimulusListSearch || "")
        .trim()
        .toLowerCase();
      if (!q) return currentStimulusAllItems.slice();
      return currentStimulusAllItems.filter(function (it) {
        return String((it && it.title) || "")
          .toLowerCase()
          .includes(q);
      });
    }

    function renderStimuliRows(rows) {
      if (!stimulusBody) return;
      if (!rows.length) {
        stimulusBody.innerHTML =
          '<tr><td colspan="6" class="muted">Tidak ada data stimuli.</td></tr>';
        return;
      }
      stimulusBody.innerHTML = "";
      rows.forEach(function (it) {
        const subject = subjectsCache.find(function (x) {
          return Number(x.id) === Number(it.subject_id);
        });
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" +
            escapeHtml(subject ? String(subject.name || "") : "-") +
            "</td>",
          "<td>" + escapeHtml(String(it.title || "")) + "</td>",
          "<td>" + escapeHtml(String(it.stimulus_type || "")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.created_at)) + "</td>",
          '<td><span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-action="preview-stimulus" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Preview" aria-label="Preview">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12z"/><circle cx="12" cy="12" r="3"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn" type="button" data-action="edit-stimulus" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Ubah" aria-label="Ubah">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 17.25V21h3.75L17.8 9.95l-3.75-3.75L3 17.25z"/><path d="M14.05 6.2l3.75 3.75"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-action="delete-stimulus" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Hapus" aria-label="Hapus">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg>' +
            "</button>" +
            "</span></td>",
        ].join("");
        stimulusBody.appendChild(tr);
      });
    }

    function updateStimulusPaginationInfo(totalRows, pageRows) {
      if (!stimulusPageInfo) return;
      if (totalRows <= 0 || pageRows <= 0) {
        stimulusPageInfo.textContent = "Baris 0 - 0";
        return;
      }
      const from = (stimulusListPage - 1) * stimulusListPageSize + 1;
      const to = from + pageRows - 1;
      stimulusPageInfo.textContent =
        "Baris " +
        String(from) +
        " - " +
        String(to) +
        " dari " +
        String(totalRows);
    }

    function renderStimuli(items) {
      currentStimulusAllItems = Array.isArray(items) ? items : [];
      const filtered = getFilteredStimulusItems();
      const total = filtered.length;
      const totalPages = Math.max(
        1,
        Math.ceil(total / Math.max(1, stimulusListPageSize)),
      );
      if (stimulusListPage > totalPages) stimulusListPage = totalPages;
      if (stimulusListPage < 1) stimulusListPage = 1;
      const start = (stimulusListPage - 1) * stimulusListPageSize;
      currentStimulusItems = filtered.slice(
        start,
        start + stimulusListPageSize,
      );
      renderStimuliRows(currentStimulusItems);
      updateStimulusPaginationInfo(total, currentStimulusItems.length);
      if (stimulusPrevBtn) stimulusPrevBtn.disabled = stimulusListPage <= 1;
      if (stimulusNextBtn)
        stimulusNextBtn.disabled = stimulusListPage >= totalPages;
    }

    function renderManuscriptRows(items) {
      if (!manuscriptBody) return;
      if (!Array.isArray(items) || items.length === 0) {
        manuscriptBody.innerHTML =
          '<tr><td colspan="7" class="muted">Belum ada data versi naskah.</td></tr>';
        return;
      }
      manuscriptBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.version_no || "")) + "</td>",
          "<td>" + escapeHtml(String(it.stimulus_id || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.status || "")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.created_at)) + "</td>",
        ].join("");

        const tdAction = document.createElement("td");
        const actionWrap = document.createElement("span");
        actionWrap.className = "action-icons";
        const qid = Number((it && it.question_id) || 0);
        const ver = Number((it && it.version_no) || 0);
        const iconMap = {
          "preview-manuscript":
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12z"/><circle cx="12" cy="12" r="3"/></svg>',
          "edit-manuscript":
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 17.25V21h3.75L17.8 9.95l-3.75-3.75L3 17.25z"/><path d="M14.05 6.2l3.75 3.75"/></svg>',
          "delete-manuscript":
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg>',
          "finalize-manuscript":
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M20 6L9 17l-5-5"/></svg>',
        };
        [
          { action: "preview-manuscript", title: "Lihat", danger: false },
          { action: "edit-manuscript", title: "Ubah", danger: false },
          { action: "delete-manuscript", title: "Hapus", danger: true },
          { action: "finalize-manuscript", title: "Finalize", danger: false },
        ].forEach(function (cfg) {
          const btn = document.createElement("button");
          btn.type = "button";
          btn.className = cfg.danger ? "icon-only-btn danger" : "icon-only-btn";
          btn.setAttribute("data-action", cfg.action);
          btn.setAttribute("data-question-id", String(qid));
          btn.setAttribute("data-version-no", String(ver));
          btn.setAttribute("title", cfg.title);
          btn.setAttribute("aria-label", cfg.title);
          btn.innerHTML = iconMap[cfg.action] || "";
          actionWrap.appendChild(btn);
        });
        tdAction.appendChild(actionWrap);
        tr.appendChild(tdAction);
        manuscriptBody.appendChild(tr);
      });
    }

    function updateManuscriptPaginationInfo(totalRows, pageRows) {
      if (!manuscriptPageInfo) return;
      if (totalRows <= 0 || pageRows <= 0) {
        manuscriptPageInfo.textContent = "Baris 0 - 0";
        return;
      }
      const from = (manuscriptPage - 1) * manuscriptPageSize + 1;
      const to = from + pageRows - 1;
      manuscriptPageInfo.textContent =
        "Baris " +
        String(from) +
        " - " +
        String(to) +
        " dari " +
        String(totalRows);
    }

    function renderManuscripts(items) {
      if (!manuscriptBody) return;
      manuscriptAllItems = Array.isArray(items) ? items.slice() : [];
      manuscriptVersionsCache = manuscriptAllItems.slice();
      const total = manuscriptAllItems.length;
      const totalPages = Math.max(
        1,
        Math.ceil(total / Math.max(1, manuscriptPageSize)),
      );
      if (manuscriptPage > totalPages) manuscriptPage = totalPages;
      if (manuscriptPage < 1) manuscriptPage = 1;
      const start = (manuscriptPage - 1) * manuscriptPageSize;
      const pageRows = manuscriptAllItems.slice(
        start,
        start + manuscriptPageSize,
      );
      renderManuscriptRows(pageRows);
      updateManuscriptPaginationInfo(total, pageRows.length);
      if (manuscriptPrevBtn) manuscriptPrevBtn.disabled = manuscriptPage <= 1;
      if (manuscriptNextBtn)
        manuscriptNextBtn.disabled = manuscriptPage >= totalPages;
    }

    async function reloadManuscriptsByCurrentTableScope(fallbackQuestionID) {
      const questionIDs = Array.from(
        new Set(
          (Array.isArray(manuscriptVersionsCache)
            ? manuscriptVersionsCache
            : []
          )
            .map(function (it) {
              return Number((it && it.question_id) || 0);
            })
            .filter(function (v) {
              return v > 0;
            }),
        ),
      );
      if (!questionIDs.length) {
        const qid = Number(
          fallbackQuestionID ||
            (manuscriptListQuestionSelect &&
              manuscriptListQuestionSelect.value) ||
            0,
        );
        if (qid <= 0) {
          renderManuscripts([]);
          return;
        }
        const one = await api("/api/v1/questions/" + qid + "/versions", "GET");
        renderManuscripts(Array.isArray(one) ? one : []);
        return;
      }
      const settled = await Promise.allSettled(
        questionIDs.map(function (qid) {
          return api("/api/v1/questions/" + qid + "/versions", "GET");
        }),
      );
      const merged = settled
        .map(function (it) {
          return it && it.status === "fulfilled" ? it.value : [];
        })
        .reduce(function (acc, rows) {
          if (Array.isArray(rows)) return acc.concat(rows);
          return acc;
        }, [])
        .sort(function (a, b) {
          const ta = new Date(a && a.created_at ? a.created_at : 0).getTime();
          const tb = new Date(b && b.created_at ? b.created_at : 0).getTime();
          if (tb !== ta) return tb - ta;
          const qa = Number((a && a.question_id) || 0);
          const qb = Number((b && b.question_id) || 0);
          if (qb !== qa) return qb - qa;
          return (
            Number((b && b.version_no) || 0) - Number((a && a.version_no) || 0)
          );
        });
      renderManuscripts(merged);
    }

    async function findManuscriptItemByKey(questionID, versionNo) {
      const qid = Number(questionID || 0);
      const ver = Number(versionNo || 0);
      if (qid <= 0 || ver <= 0) return null;
      const fromCache = (
        Array.isArray(manuscriptVersionsCache) ? manuscriptVersionsCache : []
      ).find(function (it) {
        return (
          Number((it && it.question_id) || 0) === qid &&
          Number((it && it.version_no) || 0) === ver
        );
      });
      if (fromCache) return fromCache;
      const rows = await api("/api/v1/questions/" + qid + "/versions", "GET");
      const list = Array.isArray(rows) ? rows : [];
      const found = list.find(function (it) {
        return Number((it && it.version_no) || 0) === ver;
      });
      if (found) {
        manuscriptVersionsCache = Array.isArray(manuscriptVersionsCache)
          ? manuscriptVersionsCache.concat([found])
          : [found];
        return found;
      }
      return null;
    }

    async function runManuscriptRowAction(btn, action, questionID, versionNo) {
      let item = null;
      try {
        item = await findManuscriptItemByKey(questionID, versionNo);
      } catch (err) {
        setManuscriptMsg(
          "Gagal memuat data naskah Question " +
            String(questionID) +
            " versi " +
            String(versionNo) +
            ": " +
            String((err && err.message) || "request error"),
        );
        return;
      }
      if (!item) {
        setManuscriptMsg(
          "Data naskah untuk Question " +
            String(questionID) +
            " versi " +
            String(versionNo) +
            " tidak ditemukan.",
        );
        return;
      }

      if (action === "preview-manuscript") {
        previewManuscriptVersion(item);
        return;
      }
      if (action === "edit-manuscript") {
        const status = String(item.status || "")
          .trim()
          .toLowerCase();
        if (!(status === "draft" || status === "revisi")) {
          setManuscriptMsg(
            "Versi final tidak bisa diedit. Buat draft baru untuk revisi.",
          );
          return;
        }
        const done = beginBusy(btn, msg, "Memuat naskah ke editor...");
        try {
          await beginManuscriptEdit(item);
          setManuscriptMsg(
            "Mode edit aktif untuk Question " +
              String(questionID) +
              " versi " +
              String(versionNo) +
              ".",
          );
        } catch (err) {
          setManuscriptMsg("Gagal memuat naskah untuk edit: " + err.message);
        } finally {
          done();
        }
        return;
      }
      if (action === "delete-manuscript") {
        const status = String(item.status || "")
          .trim()
          .toLowerCase();
        if (!(status === "draft" || status === "revisi")) {
          setManuscriptMsg("Versi final tidak bisa dihapus.");
          return;
        }
        const ok = window.confirm(
          "Hapus draft naskah versi " +
            String(versionNo) +
            " untuk Question " +
            String(questionID) +
            "?",
        );
        if (!ok) return;
        const done = beginBusy(btn, msg, "Menghapus draft naskah...");
        try {
          await api(
            "/api/v1/questions/" + questionID + "/versions/" + versionNo,
            "DELETE",
          );
          if (
            manuscriptEditingQuestionID === questionID &&
            Number(
              (manuscriptEditingVersionNoInput &&
                manuscriptEditingVersionNoInput.value) ||
                0,
            ) === versionNo
          ) {
            clearManuscriptEditMode();
            manuscriptForm && manuscriptForm.reset();
            resetManuscriptEditors();
          }
          await reloadManuscriptsByCurrentTableScope(questionID);
          setManuscriptMsg("Draft naskah berhasil dihapus.");
        } catch (err) {
          setManuscriptMsg("Hapus draft naskah gagal: " + err.message);
        } finally {
          done();
        }
        return;
      }
      if (action === "finalize-manuscript") {
        const status = String(item.status || "")
          .trim()
          .toLowerCase();
        if (!(status === "draft" || status === "revisi")) {
          setManuscriptMsg("Versi ini sudah final.");
          return;
        }
        const ok = window.confirm(
          "Finalize naskah versi " +
            String(versionNo) +
            " untuk Question " +
            String(questionID) +
            "?",
        );
        if (!ok) return;
        const done = beginBusy(btn, msg, "Memfinalisasi versi naskah...");
        try {
          await api(
            "/api/v1/questions/" +
              questionID +
              "/versions/" +
              versionNo +
              "/finalize",
            "POST",
          );
          if (manuscriptFinalizeQuestionSelect) {
            manuscriptFinalizeQuestionSelect.value = String(questionID);
          }
          const versionInput =
            manuscriptFinalizeForm &&
            manuscriptFinalizeForm.elements.namedItem("version_no");
          if (versionInput) versionInput.value = String(versionNo);
          await reloadManuscriptsByCurrentTableScope(questionID);
          setManuscriptMsg("Versi naskah berhasil difinalkan.");
        } catch (err) {
          setManuscriptMsg("Finalize naskah gagal: " + err.message);
        } finally {
          done();
        }
      }
    }

    async function loadSubjects() {
      const items = await api("/api/v1/subjects", "GET");
      const subjectItems = Array.isArray(items) ? items : [];
      subjectsCache = subjectItems;
      const levels = {};
      subjectItems.forEach(function (it) {
        const lvl = String(it.education_level || "").trim();
        if (lvl) levels[lvl] = true;
      });

      if (statSubjects) statSubjects.textContent = String(subjectItems.length);
      if (statLevels)
        statLevels.textContent = String(Object.keys(levels).length);

      if (!subjectsBody) return subjectItems;
      if (!subjectItems.length) {
        subjectsBody.innerHTML =
          '<tr><td colspan="5" class="muted">Tidak ada data mapel.</td></tr>';
        return subjectItems;
      }
      subjectsBody.innerHTML = "";
      subjectItems.forEach(function (it) {
        const actionHTML = canManageSubjects
          ? '<span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-action="edit-subject" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Ubah" aria-label="Ubah">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 17.25V21h3.75L17.8 9.95l-3.75-3.75L3 17.25z"/><path d="M14.05 6.2l3.75 3.75"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-action="delete-subject" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Hapus" aria-label="Hapus">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg>' +
            "</button>" +
            "</span>"
          : "-";
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.education_level || "")) + "</td>",
          "<td>" + escapeHtml(String(it.subject_type || "")) + "</td>",
          "<td>" + escapeHtml(String(it.name || "")) + "</td>",
          "<td>" + actionHTML + "</td>",
        ].join("");
        subjectsBody.appendChild(tr);
      });

      fillSubjectSelect(blueprintSubjectSelect, false);
      fillSubjectSelect(blueprintFilterSubject, true);
      fillSubjectSelect(stimulusSubjectSelect, false);
      fillSubjectSelect(stimulusListSubject, false);
      fillSubjectSelect(examSubjectSelect, false);
      return subjectItems;
    }

    async function loadBlueprints(subjectID) {
      const id = Number(subjectID || 0);
      const params = new URLSearchParams();
      if (id > 0) params.set("subject_id", String(id));
      params.set(
        "owner_only",
        isOwnerOnlyQuestionsEnabled() ? "true" : "false",
      );
      const qs = params.toString() ? "?" + params.toString() : "";
      const items = await api("/api/v1/questions" + qs, "GET");
      blueprintsCache = Array.isArray(items) ? items : [];
      renderBlueprints(blueprintsCache);
      fillBlueprintSelect(manuscriptQuestionSelect);
      fillBlueprintSelect(manuscriptListQuestionSelect);
      fillBlueprintSelect(manuscriptFinalizeQuestionSelect);
      applyManuscriptTypeUI(true);
      return blueprintsCache;
    }

    async function ensureStimuliBySubject(subjectID) {
      const id = Number(subjectID || 0);
      if (id <= 0) return [];
      if (Array.isArray(stimuliBySubject[id])) return stimuliBySubject[id];
      const items = await api(
        "/api/v1/stimuli?subject_id=" + encodeURIComponent(String(id)),
        "GET",
      );
      const rows = Array.isArray(items) ? items : [];
      stimuliBySubject[id] = rows;
      return rows;
    }

    async function loadStimuliTable(subjectID) {
      const id = Number(subjectID || 0);
      if (id <= 0) {
        renderStimuli([]);
        return [];
      }
      const rows = await ensureStimuliBySubject(id);
      currentStimulusTableSubjectID = id;
      renderStimuli(rows);
      return rows;
    }

    async function reloadCurrentStimuli() {
      const subjectID = Number(
        (stimulusListSubject && stimulusListSubject.value) ||
          currentStimulusTableSubjectID,
      );
      if (subjectID > 0) delete stimuliBySubject[subjectID];
      await loadStimuliTable(subjectID);
      await refreshStimuliSelectByQuestion();
    }

    function findStimulusByID(id) {
      return currentStimulusAllItems.find(function (it) {
        return Number(it.id) === Number(id);
      });
    }

    function renderStimulusPreviewHTML(stimulus) {
      if (!stimulus || typeof stimulus !== "object") {
        return '<p class="muted">Data stimulus tidak ditemukan.</p>';
      }
      const content = stimulus.content || {};
      if (String(stimulus.stimulus_type || "") === "multiteks") {
        const tabs = Array.isArray(content.tabs) ? content.tabs : [];
        if (!tabs.length) return '<p class="muted">Konten tab kosong.</p>';
        return tabs
          .map(function (tab, idx) {
            const title = String((tab && tab.title) || "").trim();
            const body = String((tab && tab.body) || "").trim();
            return [
              '<section class="panel mt">',
              "<h4>" + escapeHtml(title || "Tab " + String(idx + 1)) + "</h4>",
              '<div class="box mt">' +
                (body || '<span class="muted">(konten kosong)</span>') +
                "</div>",
              "</section>",
            ].join("");
          })
          .join("");
      }
      const body = String(content.body || "").trim();
      return (
        '<div class="box mt">' +
        (body || '<span class="muted">(konten kosong)</span>') +
        "</div>"
      );
    }

    async function refreshStimuliSelectByQuestion() {
      const questionID = Number(
        (manuscriptQuestionSelect && manuscriptQuestionSelect.value) || 0,
      );
      const blueprint = blueprintsCache.find(function (it) {
        return Number(it.id) === questionID;
      });
      const subjectID = Number((blueprint && blueprint.subject_id) || 0);
      const items = await ensureStimuliBySubject(subjectID);
      fillStimulusSelect(manuscriptStimulusSelect, items);
      if (manuscriptStimulusSelect) {
        manuscriptStimulusSelect.disabled = subjectID <= 0;
      }
      return items;
    }

    async function loadReviewStats() {
      const [allItems, pendingItems] = await Promise.all([
        api("/api/v1/reviews/tasks", "GET"),
        api("/api/v1/reviews/tasks?status=menunggu_reviu", "GET"),
      ]);
      const total = Array.isArray(allItems) ? allItems.length : 0;
      const pending = Array.isArray(pendingItems) ? pendingItems.length : 0;
      if (statReviewTotal) statReviewTotal.textContent = String(total);
      if (statReviewPending) statReviewPending.textContent = String(pending);
    }

    async function loadGuruExamSubjectOptions() {
      if (!examSubjectSelect) return;
      const subjects = await api("/api/v1/subjects", "GET");
      examSubjectSelect.innerHTML = '<option value="">Pilih mapel...</option>';
      if (!Array.isArray(subjects)) return;
      subjects.forEach(function (it) {
        const o = document.createElement("option");
        o.value = String(it.id || "");
        o.textContent =
          String(it.education_level || "") +
          " | " +
          String(it.subject_type || "") +
          " | " +
          String(it.name || "");
        examSubjectSelect.appendChild(o);
      });
    }

    async function loadGuruExamManageTable() {
      const exams = await api("/api/v1/admin/exams/manage", "GET");
      if (!examsBody) return;
      if (!Array.isArray(exams) || !exams.length) {
        examsBody.innerHTML =
          '<tr><td colspan="7" class="muted">Belum ada data ujian.</td></tr>';
        return;
      }
      examsBody.innerHTML = "";
      exams.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" +
            escapeHtml(String(it.code || "")) +
            "<br><small>" +
            escapeHtml(String(it.title || "")) +
            "</small></td>",
          "<td>" + escapeHtml(String(it.subject_name || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.duration_minutes || 0)) + " menit</td>",
          "<td>" + escapeHtml(String(it.question_count || 0)) + "</td>",
          "<td>" + escapeHtml(String(it.assigned_count || 0)) + "</td>",
          '<td><span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-exam-action="use" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Pilih Ujian" aria-label="Pilih Ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M5 12h14M13 6l6 6-6 6"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-exam-action="delete" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Nonaktifkan Ujian" aria-label="Nonaktifkan Ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/></svg>' +
            "</button>" +
            "</span></td>",
        ].join("");
        examsBody.appendChild(tr);
      });
    }

    async function loadGuruExamQuestions(examID) {
      if (!examQuestionsBody) return;
      if (!examID || examID <= 0) {
        examQuestionsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Pilih exam lalu muat soal ujian.</td></tr>';
        return;
      }
      const items = await api(
        "/api/v1/admin/exams/" + Number(examID) + "/questions",
        "GET",
      );
      if (!Array.isArray(items) || !items.length) {
        examQuestionsBody.innerHTML =
          '<tr><td colspan="6" class="muted">Belum ada soal di ujian ini.</td></tr>';
        return;
      }
      examQuestionsBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.seq_no || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_type || "")) + "</td>",
          "<td>" + escapeHtml(String(it.stem_preview || "")) + "</td>",
          "<td>" + escapeHtml(String(it.weight || 1)) + "</td>",
          '<td><button class="icon-only-btn danger" type="button" data-exam-question-del="' +
            escapeHtml(String(it.question_id || "")) +
            '" title="Hapus dari ujian" aria-label="Hapus dari ujian">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/></svg>' +
            "</button></td>",
        ].join("");
        examQuestionsBody.appendChild(tr);
      });
    }

    async function loadGuruExamAssignments(examID) {
      if (!resultsAssignmentBody) return;
      if (!examID || examID <= 0) {
        resultsAssignmentBody.innerHTML =
          '<tr><td colspan="6" class="muted">Isi Exam ID lalu muat peserta ujian.</td></tr>';
        return;
      }
      const items = await api(
        "/api/v1/admin/exams/" + Number(examID) + "/assignments",
        "GET",
      );
      if (!Array.isArray(items) || !items.length) {
        resultsAssignmentBody.innerHTML =
          '<tr><td colspan="6" class="muted">Belum ada peserta terdaftar.</td></tr>';
        return;
      }
      resultsAssignmentBody.innerHTML = "";
      items.forEach(function (it) {
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.user_id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.full_name || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.username || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.role || "-")) + "</td>",
          "<td>" + escapeHtml(String(it.status || "-")) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.assigned_at)) + "</td>",
        ].join("");
        resultsAssignmentBody.appendChild(tr);
      });
    }

    async function loadReviews(statusValue) {
      const status = String(statusValue || "").trim();
      const qs = status ? "?status=" + encodeURIComponent(status) : "";
      const items = await api("/api/v1/reviews/tasks" + qs, "GET");
      const reviewItems = Array.isArray(items) ? items : [];

      if (!reviewBody) return reviewItems;
      if (!reviewItems.length) {
        reviewBody.innerHTML =
          '<tr><td colspan="7" class="muted">Tidak ada data tugas reviu.</td></tr>';
        return reviewItems;
      }
      reviewBody.innerHTML = "";
      reviewItems.forEach(function (it) {
        const status = String(it.status || "");
        const canDecide = status === "menunggu_reviu";
        const actionHTML = canDecide
          ? '<span class="action-icons">' +
            '<button class="icon-only-btn" type="button" data-action="approve" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Setujui" aria-label="Setujui">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M5 13l4 4L19 7"/></svg>' +
            "</button>" +
            '<button class="icon-only-btn danger" type="button" data-action="revise" data-id="' +
            escapeHtml(String(it.id || "")) +
            '" title="Perlu Revisi" aria-label="Perlu Revisi">' +
            '<svg viewBox="0 0 24 24" focusable="false"><path d="M3 17.25V21h3.75L17.8 9.95l-3.75-3.75L3 17.25z"/><path d="M14.05 6.2l3.75 3.75"/></svg>' +
            "</button>" +
            "</span>"
          : "-";
        const tr = document.createElement("tr");
        tr.innerHTML = [
          "<td>" + escapeHtml(String(it.id || "")) + "</td>",
          "<td>" + escapeHtml(String(it.question_id || "")) + "</td>",
          "<td>" + escapeHtml(status) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.assigned_at)) + "</td>",
          "<td>" + escapeHtml(fmtDate(it.reviewed_at)) + "</td>",
          "<td>" + escapeHtml(String(it.note || "-")) + "</td>",
          "<td>" + actionHTML + "</td>",
        ].join("");
        reviewBody.appendChild(tr);
      });
      return reviewItems;
    }

    async function loadGuruDashboard() {
      const statusEl =
        reviewFilterForm && reviewFilterForm.elements.namedItem("status");
      const statusValue = statusEl ? statusEl.value : "";
      await Promise.all([
        loadEducationLevelsForGuruSubjectForm(),
        loadSubjects(),
        loadReviews(statusValue),
        loadReviewStats(),
        loadGuruExamSubjectOptions(),
        loadGuruExamManageTable(),
      ]);
      const defaultSubjectID = Number(
        (stimulusListSubject && stimulusListSubject.value) || 0,
      );
      await Promise.all([
        loadBlueprints(
          Number((blueprintFilterSubject && blueprintFilterSubject.value) || 0),
        ),
        loadStimuliTable(defaultSubjectID),
      ]);
      try {
        const initialVersions = await loadAllManuscriptVersionsFromBlueprints();
        renderManuscripts(initialVersions);
      } catch (_) {
        renderManuscripts([]);
      }
      await refreshStimuliSelectByQuestion();
      fillSubjectEducationLevelSelect("");
    }

    try {
      await ensureQuillReady();
      if (stimulusSingleEditor && !stimulusSingleQuill) {
        stimulusSingleQuill = createQuillEditor(
          stimulusSingleEditor,
          "Tulis konten stimulus...",
        );
        if (stimulusSingleQuill) {
          stimulusSingleQuill.on("text-change", function () {
            renderLiveStimulusPreview();
            clearStimulusInlineError();
          });
        }
      }
      if (stimulusPageSizeSelect) {
        stimulusListPageSize = Math.max(
          1,
          Number(stimulusPageSizeSelect.value) || 10,
        );
      }
      if (manuscriptPageSizeSelect) {
        manuscriptPageSize = Math.max(
          1,
          Number(manuscriptPageSizeSelect.value) || 15,
        );
      }
      if (questionOwnerScopeSelect) {
        const role = String((user && user.role) || "")
          .trim()
          .toLowerCase();
        questionOwnerScopeSelect.value = role === "admin" ? "all" : "mine";
      }
      if (manuscriptStemEditor && !manuscriptStemQuill) {
        manuscriptStemQuill = createQuillEditor(
          manuscriptStemEditor,
          "Tulis naskah soal...",
        );
        if (manuscriptStemQuill) {
          manuscriptStemQuill.on("text-change", function () {
            syncManuscriptHiddenInputs();
          });
        }
      }
      if (manuscriptExplanationEditor && !manuscriptExplanationQuill) {
        manuscriptExplanationQuill = createQuillEditor(
          manuscriptExplanationEditor,
          "Tulis penjelasan...",
        );
        if (manuscriptExplanationQuill) {
          manuscriptExplanationQuill.on("text-change", function () {
            syncManuscriptHiddenInputs();
          });
        }
      }
      if (manuscriptHintEditor && !manuscriptHintQuill) {
        manuscriptHintQuill = createQuillEditor(
          manuscriptHintEditor,
          "Tulis hint...",
        );
        if (manuscriptHintQuill) {
          manuscriptHintQuill.on("text-change", function () {
            syncManuscriptHiddenInputs();
          });
        }
      }
      syncManuscriptHiddenInputs();
      switchStimulusEditorByType();
      resetStimulusEditors();
      const firstMenu =
        (menuButtons &&
          menuButtons.length > 0 &&
          menuButtons[0].getAttribute("data-guru-menu")) ||
        "dashboard";
      activateGuruMenu(firstMenu);
      await loadGuruDashboard();
      renderLiveStimulusPreview();
      setMsg("Dashboard guru siap.");
    } catch (err) {
      setMsg("Gagal memuat dashboard guru: " + err.message);
    }

    menuButtons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        activateGuruMenu(btn.getAttribute("data-guru-menu") || "dashboard");
      });
    });
    root.querySelectorAll("[data-guru-jump]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        activateGuruMenu(btn.getAttribute("data-guru-jump") || "dashboard");
      });
    });

    if (stimulusTypeSelect) {
      stimulusTypeSelect.addEventListener("change", function () {
        switchStimulusEditorByType();
        if (currentStimulusType() === "multiteks" && stimulusTabsContainer) {
          if (!stimulusTabsContainer.querySelector(".stimulus-tab-item")) {
            buildStimulusTabEditor();
          }
        }
        renderLiveStimulusPreview();
        clearStimulusInlineError();
      });
    }

    if (stimulusAddTabBtn) {
      stimulusAddTabBtn.addEventListener("click", function () {
        buildStimulusTabEditor();
        renderLiveStimulusPreview();
        clearStimulusInlineError();
      });
    }

    if (stimulusForm) {
      stimulusForm.addEventListener("click", function (e) {
        const btn = e.target && e.target.closest("button[data-editor-action]");
        if (!btn) return;
        const action = String(btn.getAttribute("data-editor-action") || "");
        if (action !== "remove-tab") return;
        e.preventDefault();
        const row = btn.closest(".stimulus-tab-item");
        if (!row || !stimulusTabsContainer) return;
        row.remove();
        if (!stimulusTabsContainer.querySelector(".stimulus-tab-item")) {
          buildStimulusTabEditor();
        }
        renderLiveStimulusPreview();
        clearStimulusInlineError();
      });
    }

    if (stimulusForm) {
      stimulusForm.addEventListener("input", function () {
        renderLiveStimulusPreview();
        clearStimulusInlineError();
      });
      stimulusForm.addEventListener("change", function () {
        renderLiveStimulusPreview();
        clearStimulusInlineError();
      });
    }

    if (stimulusTabsContainer) {
      stimulusTabsContainer.addEventListener("input", function () {
        renderLiveStimulusPreview();
      });
    }

    if (stimulusBody) {
      stimulusBody.addEventListener("click", function (e) {
        const btn =
          e.target && e.target.closest("button[data-action][data-id]");
        if (!btn) return;
        const action = String(btn.getAttribute("data-action") || "");
        const id = Number(btn.getAttribute("data-id") || 0);
        if (!id) return;
        const item = findStimulusByID(id);
        if (!item) {
          setMsg("Data stimulus tidak ditemukan.");
          return;
        }

        if (action === "preview-stimulus") {
          if (stimulusPreviewTitle) {
            text(
              stimulusPreviewTitle,
              "Preview Stimulus: " + String(item.title || "Tanpa Judul"),
            );
          }
          if (stimulusPreviewContent) {
            stimulusPreviewContent.innerHTML = renderStimulusPreviewHTML(item);
          }
          if (
            stimulusPreviewDialog &&
            typeof stimulusPreviewDialog.showModal === "function"
          ) {
            stimulusPreviewDialog.showModal();
          }
          return;
        }

        if (action === "edit-stimulus") {
          if (!stimulusEditForm) return;
          const idField = stimulusEditForm.elements.namedItem("id");
          const subjectField =
            stimulusEditForm.elements.namedItem("subject_id");
          const titleField = stimulusEditForm.elements.namedItem("title");
          const typeField =
            stimulusEditForm.elements.namedItem("stimulus_type");
          const contentField =
            stimulusEditForm.elements.namedItem("content_raw");
          fillSubjectSelect(stimulusEditSubject, false);
          if (idField) idField.value = String(item.id || "");
          if (subjectField) subjectField.value = String(item.subject_id || "");
          if (titleField) titleField.value = String(item.title || "");
          if (typeField)
            typeField.value = String(item.stimulus_type || "single");
          if (contentField) {
            contentField.value = JSON.stringify(item.content || {}, null, 2);
          }
          if (
            stimulusEditDialog &&
            typeof stimulusEditDialog.showModal === "function"
          ) {
            stimulusEditDialog.showModal();
          }
          return;
        }

        if (action === "delete-stimulus") {
          if (!stimulusDeleteForm) return;
          const idField = stimulusDeleteForm.elements.namedItem("id");
          if (idField) idField.value = String(item.id || "");
          if (stimulusDeleteLabel) {
            text(
              stimulusDeleteLabel,
              "Stimulus '" +
                String(item.title || "Tanpa Judul") +
                "' akan dihapus.",
            );
          }
          if (
            stimulusDeleteDialog &&
            typeof stimulusDeleteDialog.showModal === "function"
          ) {
            stimulusDeleteDialog.showModal();
          }
        }
      });
    }

    if (stimulusEditCancelBtn) {
      stimulusEditCancelBtn.addEventListener("click", function () {
        if (
          stimulusEditDialog &&
          typeof stimulusEditDialog.close === "function"
        ) {
          stimulusEditDialog.close();
        }
      });
    }

    if (stimulusDeleteCancelBtn) {
      stimulusDeleteCancelBtn.addEventListener("click", function () {
        if (
          stimulusDeleteDialog &&
          typeof stimulusDeleteDialog.close === "function"
        ) {
          stimulusDeleteDialog.close();
        }
      });
    }

    if (stimulusEditForm) {
      stimulusEditForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusEditForm);
        const id = Number(fd.get("id") || 0);
        const done = beginBusy(
          stimulusEditForm,
          msg,
          "Menyimpan perubahan stimulus...",
        );
        try {
          await api("/api/v1/stimuli/" + id, "PUT", {
            subject_id: Number(fd.get("subject_id") || 0),
            title: String(fd.get("title") || "").trim(),
            stimulus_type: String(fd.get("stimulus_type") || "").trim(),
            content: JSON.parse(String(fd.get("content_raw") || "{}")),
          });
          if (
            stimulusEditDialog &&
            typeof stimulusEditDialog.close === "function"
          ) {
            stimulusEditDialog.close();
          }
          await reloadCurrentStimuli();
          setMsg("Stimulus berhasil diperbarui.");
        } catch (err) {
          setMsg("Ubah stimulus gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusDeleteForm) {
      stimulusDeleteForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusDeleteForm);
        const id = Number(fd.get("id") || 0);
        const done = beginBusy(
          stimulusDeleteForm,
          msg,
          "Menghapus stimulus...",
        );
        try {
          await api("/api/v1/stimuli/" + id, "DELETE");
          if (
            stimulusDeleteDialog &&
            typeof stimulusDeleteDialog.close === "function"
          ) {
            stimulusDeleteDialog.close();
          }
          await reloadCurrentStimuli();
          setMsg("Stimulus berhasil dihapus.");
        } catch (err) {
          setMsg("Hapus stimulus gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (blueprintForm) {
      blueprintForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(blueprintForm);
        const done = beginBusy(blueprintForm, msg, "Menyimpan kisi-kisi...");
        try {
          const payload = {
            subject_id: Number(fd.get("subject_id") || 0),
            question_type: String(fd.get("question_type") || "").trim(),
            title: String(fd.get("title") || "").trim(),
            indicator: String(fd.get("indicator") || "").trim(),
            material: String(fd.get("material") || "").trim(),
            objective: String(fd.get("objective") || "").trim(),
            cognitive_level: String(fd.get("cognitive_level") || "").trim(),
          };
          await api("/api/v1/questions", "POST", payload);
          await loadBlueprints(
            Number(
              (blueprintFilterSubject && blueprintFilterSubject.value) || 0,
            ),
          );
          await refreshStimuliSelectByQuestion();
          blueprintForm.reset();
          fillSubjectSelect(blueprintSubjectSelect, false);
          setMsg("Kisi-kisi berhasil disimpan.");
        } catch (err) {
          setMsg("Simpan kisi-kisi gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (blueprintFilterForm) {
      blueprintFilterForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const done = beginBusy(blueprintFilterForm, msg, "Memuat kisi-kisi...");
        try {
          await loadBlueprints(
            Number(
              (blueprintFilterSubject && blueprintFilterSubject.value) || 0,
            ),
          );
          await refreshStimuliSelectByQuestion();
          setMsg("Data kisi-kisi berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat kisi-kisi: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (blueprintRefreshBtn) {
      blueprintRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          blueprintRefreshBtn,
          msg,
          "Muat ulang kisi-kisi...",
        );
        try {
          await loadBlueprints(
            Number(
              (blueprintFilterSubject && blueprintFilterSubject.value) || 0,
            ),
          );
          await refreshStimuliSelectByQuestion();
          setMsg("Kisi-kisi berhasil dimuat ulang.");
        } catch (err) {
          setMsg("Gagal muat ulang kisi-kisi: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusForm) {
      stimulusForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusForm);
        const content = collectStimulusContentPayload();
        const errors = validateStimulusFormPayload(fd, content);
        if (errors.length) {
          showStimulusInlineError(errors);
          setMsg("Validasi gagal. Periksa form stimulus.");
          return;
        }
        const done = beginBusy(stimulusForm, msg, "Menyimpan stimulus...");
        try {
          const subjectID = Number(fd.get("subject_id") || 0);
          const stimulusType = String(fd.get("stimulus_type") || "").trim();
          const payload = {
            subject_id: subjectID,
            title: String(fd.get("title") || "").trim(),
            stimulus_type: stimulusType,
            content: content,
          };
          await api("/api/v1/stimuli", "POST", payload);
          delete stimuliBySubject[subjectID];
          await loadStimuliTable(subjectID);
          await refreshStimuliSelectByQuestion();
          stimulusForm.reset();
          clearStimulusInlineError();
          fillSubjectSelect(stimulusSubjectSelect, false);
          switchStimulusEditorByType();
          resetStimulusEditors();
          renderLiveStimulusPreview();
          setMsg("Stimulus berhasil disimpan.");
        } catch (err) {
          showStimulusInlineError(String(err.message || "Simpan gagal."));
          setMsg("Simpan stimulus gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusListForm) {
      stimulusListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        stimulusListSearch = String(
          (stimulusSearchInput && stimulusSearchInput.value) || "",
        ).trim();
        stimulusListPageSize = Math.max(
          1,
          Number(
            (stimulusPageSizeSelect && stimulusPageSizeSelect.value) || 10,
          ),
        );
        stimulusListPage = 1;
        const done = beginBusy(stimulusListForm, msg, "Memuat stimuli...");
        try {
          const subjectID = Number(
            (stimulusListSubject && stimulusListSubject.value) || 0,
          );
          await loadStimuliTable(subjectID);
          setMsg("Data stimuli berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat stimuli: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusSearchInput) {
      stimulusSearchInput.addEventListener("input", function () {
        stimulusListSearch = String(stimulusSearchInput.value || "").trim();
        stimulusListPage = 1;
        renderStimuli(currentStimulusAllItems);
      });
    }

    if (stimulusPageSizeSelect) {
      stimulusPageSizeSelect.addEventListener("change", function () {
        stimulusListPageSize = Math.max(
          1,
          Number(stimulusPageSizeSelect.value) || 10,
        );
        stimulusListPage = 1;
        renderStimuli(currentStimulusAllItems);
      });
    }

    if (stimulusPrevBtn) {
      stimulusPrevBtn.addEventListener("click", function () {
        stimulusListPage = Math.max(1, stimulusListPage - 1);
        renderStimuli(currentStimulusAllItems);
      });
    }

    if (stimulusNextBtn) {
      stimulusNextBtn.addEventListener("click", function () {
        stimulusListPage += 1;
        renderStimuli(currentStimulusAllItems);
      });
    }

    if (stimulusTemplateBtn) {
      stimulusTemplateBtn.addEventListener("click", async function () {
        const done = beginBusy(
          stimulusTemplateBtn,
          msg,
          "Menyiapkan template stimuli...",
        );
        try {
          await downloadStimulusTemplateCSV();
          setMsg("Template import stimuli berhasil diunduh.");
        } catch (err) {
          setMsg("Gagal mengunduh template stimuli: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusImportForm) {
      stimulusImportForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const file =
          stimulusImportFileInput &&
          stimulusImportFileInput.files &&
          stimulusImportFileInput.files[0];
        if (!file) {
          setMsg("Pilih file CSV terlebih dahulu.");
          return;
        }
        const fd = new FormData();
        fd.append("file", file);
        const done = beginBusy(
          stimulusImportForm,
          msg,
          "Mengimpor stimuli dari CSV...",
        );
        try {
          const out = await apiMultipart("/api/v1/stimuli/import", "POST", fd);
          const report = (out && out.report) || {};
          const total = Number(report.total_rows || 0);
          const okRows = Number(report.success_rows || 0);
          const failRows = Number(report.failed_rows || 0);
          const errors = Array.isArray(report.errors) ? report.errors : [];
          lastStimulusImportErrors = errors;
          const details = formatStimulusImportErrors(report.errors);
          if (failRows > 0) {
            if (stimulusImportErrorOutput) {
              stimulusImportErrorOutput.textContent =
                formatStimulusImportErrorsMultiline(errors);
            }
            if (
              stimulusImportErrorDialog &&
              typeof stimulusImportErrorDialog.showModal === "function"
            ) {
              stimulusImportErrorDialog.showModal();
            }
            setMsg(
              "Import stimuli selesai. Total: " +
                String(total) +
                ", berhasil: " +
                String(okRows) +
                ", gagal: " +
                String(failRows) +
                (details ? " | " + details : ""),
            );
          } else {
            setMsg(
              "Import stimuli berhasil. Total: " +
                String(total) +
                ", berhasil: " +
                String(okRows),
            );
          }
          stimulusImportForm.reset();
          await loadStimuliTable(
            Number(
              (stimulusListSubject && stimulusListSubject.value) ||
                currentStimulusTableSubjectID,
            ),
          );
          await refreshStimuliSelectByQuestion();
        } catch (err) {
          lastStimulusImportErrors = [];
          if (stimulusImportErrorOutput) {
            stimulusImportErrorOutput.textContent =
              "Import gagal:\n" + String(err.message || "Request failed");
          }
          if (
            stimulusImportErrorDialog &&
            typeof stimulusImportErrorDialog.showModal === "function"
          ) {
            stimulusImportErrorDialog.showModal();
          }
          setMsg("Import stimuli gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (stimulusImportDownloadBtn) {
      stimulusImportDownloadBtn.addEventListener("click", function () {
        if (!lastStimulusImportErrors.length) {
          setMsg("Tidak ada data error untuk diunduh.");
          return;
        }
        downloadStimulusImportErrorsCSV(lastStimulusImportErrors);
        setMsg("Laporan gagal import stimuli berhasil diunduh.");
      });
    }

    if (stimulusImportErrorCloseBtn) {
      stimulusImportErrorCloseBtn.addEventListener("click", function () {
        if (
          stimulusImportErrorDialog &&
          typeof stimulusImportErrorDialog.close === "function"
        ) {
          stimulusImportErrorDialog.close();
        }
      });
    }

    if (stimulusRefreshBtn) {
      stimulusRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          stimulusRefreshBtn,
          msg,
          "Muat ulang stimuli...",
        );
        try {
          const subjectID = Number(
            (stimulusListSubject && stimulusListSubject.value) ||
              currentStimulusTableSubjectID,
          );
          if (subjectID > 0) delete stimuliBySubject[subjectID];
          await loadStimuliTable(subjectID);
          await refreshStimuliSelectByQuestion();
          setMsg("Stimuli berhasil dimuat ulang.");
        } catch (err) {
          setMsg("Gagal muat ulang stimuli: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptQuestionSelect) {
      manuscriptQuestionSelect.addEventListener("change", async function () {
        clearManuscriptEditMode();
        applyManuscriptTypeUI(true);
        try {
          await refreshStimuliSelectByQuestion();
        } catch (err) {
          setMsg("Gagal memuat stimulus naskah: " + err.message);
        }
      });
    }

    if (manuscriptQuestionTypeUI) {
      manuscriptQuestionTypeUI.addEventListener("change", function () {
        applyManuscriptTypeUI(false);
      });
    }

    if (manuscriptForm) {
      manuscriptForm.addEventListener("change", function (e) {
        const target = e.target;
        if (!target) return;
        if (
          target.name === "manuscript_mc_key" ||
          target.name === "manuscript_mr_key" ||
          target.hasAttribute("data-option-key") ||
          target.hasAttribute("data-option-html") ||
          target.hasAttribute("data-statement-id") ||
          target.hasAttribute("data-statement-text") ||
          target.hasAttribute("data-statement-correct")
        ) {
          if (
            target.hasAttribute("data-option-key") ||
            target.hasAttribute("data-option-html")
          ) {
            renderManuscriptKeyChoices();
          }
          syncManuscriptAnswerKeyPreview();
        }
      });
      manuscriptForm.addEventListener("input", function (e) {
        const target = e.target;
        if (!target) return;
        if (
          target.hasAttribute("data-option-key") ||
          target.hasAttribute("data-option-html") ||
          target.hasAttribute("data-statement-id") ||
          target.hasAttribute("data-statement-text")
        ) {
          if (
            target.hasAttribute("data-option-key") ||
            target.hasAttribute("data-option-html")
          ) {
            renderManuscriptKeyChoices();
          }
          syncManuscriptAnswerKeyPreview();
        }
      });
    }

    if (manuscriptAddOptionBtn) {
      manuscriptAddOptionBtn.addEventListener("click", function () {
        if (!manuscriptOptionsWrap) return;
        const idx =
          manuscriptOptionsWrap.querySelectorAll("[data-option-row]").length ||
          0;
        manuscriptOptionsWrap.insertAdjacentHTML(
          "beforeend",
          createManuscriptOptionRow({ option_key: "", option_html: "" }, idx),
        );
        bindManuscriptOptionEditors();
        renderManuscriptKeyChoices();
        syncManuscriptAnswerKeyPreview();
      });
    }

    if (manuscriptOptionsWrap) {
      manuscriptOptionsWrap.addEventListener("click", function (e) {
        const btn = e.target && e.target.closest("button[data-option-remove]");
        if (!btn) return;
        const row = btn.closest("[data-option-row]");
        if (row) row.remove();
        ensureManuscriptOptionRows();
        renderManuscriptKeyChoices();
        syncManuscriptAnswerKeyPreview();
      });
    }

    if (manuscriptAddStatementBtn) {
      manuscriptAddStatementBtn.addEventListener("click", function () {
        if (!manuscriptStatementsWrap) return;
        const idx =
          manuscriptStatementsWrap.querySelectorAll("[data-statement-row]")
            .length || 0;
        manuscriptStatementsWrap.insertAdjacentHTML(
          "beforeend",
          createTFStatementRow(
            { id: "S" + String(idx + 1), statement: "", correct: true },
            idx,
          ),
        );
        syncManuscriptAnswerKeyPreview();
      });
    }

    if (manuscriptStatementsWrap) {
      manuscriptStatementsWrap.addEventListener("click", function (e) {
        const btn =
          e.target && e.target.closest("button[data-statement-remove]");
        if (!btn) return;
        const row = btn.closest("[data-statement-row]");
        if (row) row.remove();
        ensureTFStatementRows();
        syncManuscriptAnswerKeyPreview();
      });
    }

    if (manuscriptImportBtn) {
      manuscriptImportBtn.addEventListener("click", function () {
        fillSubjectSelect(manuscriptImportSubjectSelect, false);
        if (manuscriptImportForm) manuscriptImportForm.reset();
        if (manuscriptImportOutput) {
          manuscriptImportOutput.textContent =
            "Gunakan format TYPE / Stimulus / Q / Opsi / Kunci.";
        }
        if (
          manuscriptImportDialog &&
          typeof manuscriptImportDialog.showModal === "function"
        ) {
          manuscriptImportDialog.showModal();
        }
      });
    }

    if (manuscriptImportCancelBtn) {
      manuscriptImportCancelBtn.addEventListener("click", function () {
        if (
          manuscriptImportDialog &&
          typeof manuscriptImportDialog.close === "function"
        ) {
          manuscriptImportDialog.close();
        }
      });
    }

    if (manuscriptImportForm) {
      manuscriptImportForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const subjectID = Number(
          (manuscriptImportSubjectSelect &&
            manuscriptImportSubjectSelect.value) ||
            0,
        );
        const file =
          manuscriptImportFileInput &&
          manuscriptImportFileInput.files &&
          manuscriptImportFileInput.files[0];
        if (subjectID <= 0) {
          setMsg("Mapel import soal wajib dipilih.");
          return;
        }
        if (!file) {
          setMsg("Pilih file CSV/TXT untuk import soal.");
          return;
        }
        const done = beginBusy(
          manuscriptImportForm,
          msg,
          "Mengimpor soal dari CSV Word...",
        );
        try {
          const rawText = await file.text();
          const blocks = parseImportWordBlocks(rawText);
          if (!blocks.length) {
            throw new Error("File tidak berisi blok soal yang valid.");
          }
          const report = {
            total_blocks: blocks.length,
            success_blocks: 0,
            failed_blocks: 0,
            errors: [],
            imported_question_ids: [],
          };
          const knownStimuli = await ensureStimuliBySubject(subjectID);
          const stimulusByKey = {};
          knownStimuli.forEach(function (it) {
            if (String((it && it.stimulus_type) || "") !== "single") return;
            const key = normalizeTextKey(readStimulusBodyText(it));
            if (!key) return;
            if (!stimulusByKey[key]) stimulusByKey[key] = it;
          });
          for (let i = 0; i < blocks.length; i += 1) {
            const blockNo = i + 1;
            try {
              const parsed = parseImportQuestionBlock(blocks[i], blockNo);
              let stimulusID = null;
              if (parsed.stimulus_text) {
                const stimulusKey = normalizeTextKey(parsed.stimulus_text);
                const existingStimulus =
                  stimulusKey && stimulusByKey[stimulusKey]
                    ? stimulusByKey[stimulusKey]
                    : null;
                if (existingStimulus && Number(existingStimulus.id || 0) > 0) {
                  stimulusID = Number(existingStimulus.id || 0);
                } else {
                  const labelMap = {
                    pg_tunggal: "MC",
                    multi_jawaban: "MR",
                    benar_salah_pernyataan: "TF",
                  };
                  const autoTitle = buildAutoStimulusTitle(
                    blockNo,
                    labelMap[parsed.question_type] || "",
                  );
                  const safeStimulus = escapeHtml(parsed.stimulus_text);
                  const stim = await api("/api/v1/stimuli", "POST", {
                    subject_id: subjectID,
                    title: autoTitle,
                    stimulus_type: "single",
                    content: { body: "<p>" + safeStimulus + "</p>" },
                  });
                  const sid = Number((stim && stim.id) || 0);
                  if (sid > 0) {
                    stimulusID = sid;
                    if (stimulusKey) stimulusByKey[stimulusKey] = stim;
                  }
                }
              }
              const blueprint = await api("/api/v1/questions", "POST", {
                subject_id: subjectID,
                question_type: parsed.question_type,
                title: parsed.title,
                indicator: parsed.indicator,
                material: parsed.material,
                objective: parsed.objective,
                cognitive_level: parsed.cognitive_level,
              });
              const qid = Number((blueprint && blueprint.id) || 0);
              if (qid <= 0) {
                throw new Error("gagal membuat kisi-kisi/soal.");
              }
              await api("/api/v1/questions/" + qid + "/versions", "POST", {
                stimulus_id: stimulusID,
                stem_html: parsed.stem_html,
                answer_key: parsed.answer_key,
                options: parsed.options,
              });
              report.success_blocks += 1;
              report.imported_question_ids.push(qid);
            } catch (err) {
              report.failed_blocks += 1;
              report.errors.push({
                block: blockNo,
                error: String((err && err.message) || "error tidak diketahui"),
              });
            }
          }
          if (manuscriptImportOutput) {
            manuscriptImportOutput.textContent = JSON.stringify(
              report,
              null,
              2,
            );
          }
          await Promise.all([
            loadBlueprints(subjectID),
            loadReviewStats(),
            loadStimuliTable(subjectID),
          ]);
          if (
            manuscriptImportDialog &&
            typeof manuscriptImportDialog.close === "function"
          ) {
            manuscriptImportDialog.close();
          }
          if (manuscriptImportForm) manuscriptImportForm.reset();
          const importedQuestionIDs = Array.from(
            new Set(
              (Array.isArray(report.imported_question_ids)
                ? report.imported_question_ids
                : []
              )
                .map(function (v) {
                  return Number(v || 0);
                })
                .filter(function (v) {
                  return v > 0;
                }),
            ),
          );
          if (importedQuestionIDs.length) {
            const batches = await Promise.all(
              importedQuestionIDs.map(function (qid) {
                return api("/api/v1/questions/" + qid + "/versions", "GET");
              }),
            );
            const mergedVersions = batches
              .reduce(function (acc, items) {
                if (Array.isArray(items)) return acc.concat(items);
                return acc;
              }, [])
              .sort(function (a, b) {
                const ta = new Date(
                  a && a.created_at ? a.created_at : 0,
                ).getTime();
                const tb = new Date(
                  b && b.created_at ? b.created_at : 0,
                ).getTime();
                if (tb !== ta) return tb - ta;
                const qa = Number((a && a.question_id) || 0);
                const qb = Number((b && b.question_id) || 0);
                if (qb !== qa) return qb - qa;
                return (
                  Number((b && b.version_no) || 0) -
                  Number((a && a.version_no) || 0)
                );
              });
            renderManuscripts(mergedVersions);
            if (manuscriptListQuestionSelect) {
              manuscriptListQuestionSelect.value = String(
                importedQuestionIDs[0],
              );
            }
            if (manuscriptFinalizeQuestionSelect) {
              manuscriptFinalizeQuestionSelect.value = String(
                importedQuestionIDs[0],
              );
            }
            if (!mergedVersions.length && manuscriptListQuestionSelect) {
              const fallbackQID = Number(
                manuscriptListQuestionSelect.value || 0,
              );
              if (fallbackQID > 0) {
                const fallbackVersions = await api(
                  "/api/v1/questions/" + fallbackQID + "/versions",
                  "GET",
                );
                renderManuscripts(
                  Array.isArray(fallbackVersions) ? fallbackVersions : [],
                );
              }
            }
          }
          const maxErrorPreview = 8;
          const errorPreview = report.errors.slice(0, maxErrorPreview);
          const errorDetail = report.errors.length
            ? " Penyebab gagal: " +
              errorPreview
                .map(function (it) {
                  return (
                    "blok " +
                    String((it && it.block) || "-") +
                    " (" +
                    String((it && it.error) || "error") +
                    ")"
                  );
                })
                .join("; ") +
              (report.errors.length > maxErrorPreview
                ? "; dan " +
                  String(report.errors.length - maxErrorPreview) +
                  " error lainnya"
                : "")
            : "";
          setMsg(
            "Import soal selesai. Berhasil: " +
              String(report.success_blocks) +
              ", gagal: " +
              String(report.failed_blocks) +
              "." +
              errorDetail,
          );
        } catch (err) {
          if (manuscriptImportOutput) {
            manuscriptImportOutput.textContent =
              "Import gagal:\n" +
              String((err && err.message) || "Request failed");
          }
          setMsg("Import soal gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptForm) {
      manuscriptForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        syncManuscriptHiddenInputs();
        const fd = new FormData(manuscriptForm);
        const editingVersionNo = Number(fd.get("editing_version_no") || 0);
        const isEditMode = editingVersionNo > 0;
        const stemHTML = String(fd.get("stem_html") || "");
        if (isBlankHTML(stemHTML)) {
          setMsg("Naskah Soal (Stem) wajib diisi.");
          return;
        }
        const blueprintType = getCurrentManuscriptQuestionType();
        const selectedType = String(
          (manuscriptQuestionTypeUI && manuscriptQuestionTypeUI.value) || "",
        )
          .trim()
          .toLowerCase();
        if (!selectedType) {
          setMsg("Tipe Soal wajib dipilih.");
          return;
        }
        if (blueprintType && selectedType !== blueprintType) {
          setMsg(
            "Tipe Soal harus sama dengan tipe pada Kisi-Kisi terpilih (" +
              blueprintType +
              ").",
          );
          return;
        }
        currentManuscriptQuestionType = selectedType;
        let answerKeyObj;
        let optionsPayload = [];
        try {
          optionsPayload = validateManuscriptOptions(
            collectManuscriptOptions(),
          );
          answerKeyObj = collectManuscriptAnswerKey();
          if (manuscriptAnswerKeyRaw) {
            manuscriptAnswerKeyRaw.value = JSON.stringify(
              answerKeyObj,
              null,
              2,
            );
          }
        } catch (err) {
          setMsg("Kunci jawaban belum valid: " + err.message);
          return;
        }
        const done = beginBusy(
          manuscriptForm,
          msg,
          isEditMode
            ? "Menyimpan perubahan naskah..."
            : "Menyimpan draft naskah...",
        );
        try {
          const questionID = isEditMode
            ? manuscriptEditingQuestionID
            : Number(fd.get("question_id") || 0);
          const stimulusID = Number(fd.get("stimulus_id") || 0);
          const payload = {
            stimulus_id: stimulusID,
            stem_html: stemHTML.trim(),
            answer_key: answerKeyObj,
            options: optionsPayload,
            explanation_html: String(fd.get("explanation_html") || "").trim(),
            hint_html: String(fd.get("hint_html") || "").trim(),
          };
          const endpoint = isEditMode
            ? "/api/v1/questions/" +
              questionID +
              "/versions/" +
              editingVersionNo
            : "/api/v1/questions/" + questionID + "/versions";
          const method = isEditMode ? "PUT" : "POST";
          await api(endpoint, method, payload);
          const versions = await api(
            "/api/v1/questions/" + questionID + "/versions",
            "GET",
          );
          renderManuscripts(versions);
          if (manuscriptListQuestionSelect) {
            manuscriptListQuestionSelect.value = String(questionID);
          }
          if (manuscriptFinalizeQuestionSelect) {
            manuscriptFinalizeQuestionSelect.value = String(questionID);
          }
          resetManuscriptEditors();
          manuscriptForm.reset();
          fillBlueprintSelect(manuscriptQuestionSelect);
          fillStimulusSelect(manuscriptStimulusSelect, []);
          if (manuscriptStimulusSelect) {
            manuscriptStimulusSelect.disabled = true;
          }
          if (manuscriptStatementsWrap) {
            manuscriptStatementsWrap.innerHTML = "";
          }
          if (manuscriptOptionsWrap) {
            manuscriptOptionsWrap.innerHTML = "";
          }
          ensureManuscriptOptionRows();
          ensureTFStatementRows();
          applyManuscriptTypeUI(true);
          clearManuscriptEditMode();
          setMsg(
            isEditMode
              ? "Draft naskah soal berhasil diperbarui."
              : "Draft naskah soal berhasil disimpan.",
          );
        } catch (err) {
          setMsg(
            (isEditMode
              ? "Simpan perubahan naskah gagal: "
              : "Simpan draft naskah gagal: ") + err.message,
          );
        } finally {
          done();
        }
      });
    }

    if (manuscriptListForm) {
      manuscriptListForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const done = beginBusy(
          manuscriptListForm,
          msg,
          "Memuat versi naskah...",
        );
        try {
          const questionID = Number(
            (manuscriptListQuestionSelect &&
              manuscriptListQuestionSelect.value) ||
              0,
          );
          const versions =
            questionID > 0
              ? await api(
                  "/api/v1/questions/" + questionID + "/versions",
                  "GET",
                )
              : await loadAllManuscriptVersionsFromBlueprints();
          renderManuscripts(versions);
          setMsg("Versi naskah berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat versi naskah: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptFinalizeForm) {
      manuscriptFinalizeForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(manuscriptFinalizeForm);
        const done = beginBusy(
          manuscriptFinalizeForm,
          msg,
          "Memfinalisasi versi naskah...",
        );
        try {
          const questionID = Number(fd.get("question_id") || 0);
          const versionNo = Number(fd.get("version_no") || 0);
          await api(
            "/api/v1/questions/" +
              questionID +
              "/versions/" +
              versionNo +
              "/finalize",
            "POST",
          );
          const versions = await api(
            "/api/v1/questions/" + questionID + "/versions",
            "GET",
          );
          renderManuscripts(versions);
          setMsg("Versi naskah berhasil difinalkan.");
        } catch (err) {
          setMsg("Finalize naskah gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptRefreshBtn) {
      manuscriptRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          manuscriptRefreshBtn,
          msg,
          "Muat ulang naskah...",
        );
        try {
          const questionID = Number(
            (manuscriptListQuestionSelect &&
              manuscriptListQuestionSelect.value) ||
              0,
          );
          const versions =
            questionID > 0
              ? await api(
                  "/api/v1/questions/" + questionID + "/versions",
                  "GET",
                )
              : await loadAllManuscriptVersionsFromBlueprints();
          renderManuscripts(versions);
          setMsg("Data naskah berhasil dimuat ulang.");
        } catch (err) {
          setMsg("Gagal muat ulang naskah: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptPageSizeSelect) {
      manuscriptPageSizeSelect.addEventListener("change", function () {
        manuscriptPageSize = Math.max(
          1,
          Number(manuscriptPageSizeSelect.value) || 15,
        );
        manuscriptPage = 1;
        renderManuscripts(manuscriptAllItems);
      });
    }

    if (manuscriptPrevBtn) {
      manuscriptPrevBtn.addEventListener("click", function () {
        manuscriptPage = Math.max(1, manuscriptPage - 1);
        renderManuscripts(manuscriptAllItems);
      });
    }

    if (manuscriptNextBtn) {
      manuscriptNextBtn.addEventListener("click", function () {
        manuscriptPage += 1;
        renderManuscripts(manuscriptAllItems);
      });
    }

    if (manuscriptCancelEditBtn) {
      manuscriptCancelEditBtn.addEventListener("click", function () {
        if (!manuscriptForm) return;
        clearManuscriptEditMode();
        manuscriptForm.reset();
        resetManuscriptEditors();
        if (manuscriptStatementsWrap) manuscriptStatementsWrap.innerHTML = "";
        if (manuscriptOptionsWrap) manuscriptOptionsWrap.innerHTML = "";
        ensureManuscriptOptionRows();
        ensureTFStatementRows();
        applyManuscriptTypeUI(true);
      });
    }

    if (questionOwnerScopeSelect) {
      questionOwnerScopeSelect.addEventListener("change", async function () {
        const done = beginBusy(
          questionOwnerScopeSelect,
          msg,
          "Memuat data soal sesuai scope...",
        );
        try {
          await loadBlueprints(
            Number(
              (blueprintFilterSubject && blueprintFilterSubject.value) || 0,
            ),
          );
          const versions = await loadAllManuscriptVersionsFromBlueprints();
          manuscriptPage = 1;
          renderManuscripts(versions);
          setManuscriptMsg("Scope soal berhasil diperbarui.");
        } catch (err) {
          setManuscriptMsg("Gagal memuat scope soal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (manuscriptBody) {
      manuscriptBody.addEventListener("click", async function (e) {
        const btn =
          e.target &&
          e.target.closest(
            "button[data-action][data-question-id][data-version-no]",
          );
        if (!btn) return;
        const action = String(btn.getAttribute("data-action") || "").trim();
        if (
          action !== "preview-manuscript" &&
          action !== "edit-manuscript" &&
          action !== "delete-manuscript" &&
          action !== "finalize-manuscript"
        ) {
          return;
        }
        const questionID = Number(btn.getAttribute("data-question-id") || 0);
        const versionNo = Number(btn.getAttribute("data-version-no") || 0);
        if (questionID <= 0 || versionNo <= 0) {
          setManuscriptMsg(
            "Aksi tidak valid: question/version tidak ditemukan.",
          );
          return;
        }
        e.preventDefault();
        await runManuscriptRowAction(btn, action, questionID, versionNo);
      });
    }

    if (reviewFilterForm) {
      reviewFilterForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const done = beginBusy(reviewFilterForm, msg, "Memuat tugas reviu...");
        try {
          const statusEl = reviewFilterForm.elements.namedItem("status");
          const statusValue = statusEl ? statusEl.value : "";
          await Promise.all([loadReviews(statusValue), loadReviewStats()]);
          setMsg("Data tugas reviu berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat tugas reviu: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (reviewRefreshBtn) {
      reviewRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          reviewRefreshBtn,
          msg,
          "Memuat ulang tugas reviu...",
        );
        try {
          const statusEl =
            reviewFilterForm && reviewFilterForm.elements.namedItem("status");
          const statusValue = statusEl ? statusEl.value : "";
          await Promise.all([loadReviews(statusValue), loadReviewStats()]);
          setMsg("Tugas reviu berhasil dimuat ulang.");
        } catch (err) {
          setMsg("Gagal memuat ulang tugas reviu: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examsRefreshBtn) {
      examsRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(examsRefreshBtn, msg, "Memuat data ujian...");
        try {
          await Promise.all([
            loadGuruExamManageTable(),
            loadGuruExamSubjectOptions(),
          ]);
          setMsg("Data ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat data ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examForm) {
      examForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        if (examFormSubmitting) {
          setMsg("Penyimpanan ujian sedang diproses. Mohon tunggu...");
          return;
        }
        const fd = new FormData(examForm);
        const examID = Number(fd.get("exam_id") || 0);
        const payload = {
          code: String(fd.get("code") || "").trim(),
          title: String(fd.get("title") || "").trim(),
          subject_id: Number(fd.get("subject_id") || 0),
          duration_minutes: Number(fd.get("duration_minutes") || 90),
          review_policy: String(fd.get("review_policy") || "after_submit"),
          is_active: true,
        };
        examFormSubmitting = true;
        const done = beginBusy(examForm, msg, "Menyimpan ujian...");
        try {
          const out =
            examID > 0
              ? await api("/api/v1/admin/exams/" + examID, "PUT", payload)
              : await api("/api/v1/admin/exams", "POST", payload);
          const savedID = out && out.id ? String(out.id) : "-";
          setMsg("Ujian berhasil disimpan. ID: " + savedID);
          examForm.reset();
          const examIDInput = examForm.elements.namedItem("exam_id");
          if (examIDInput) examIDInput.value = "";
          await loadGuruExamManageTable();
        } catch (err) {
          setMsg("Gagal menyimpan ujian: " + err.message);
        } finally {
          done();
          window.setTimeout(function () {
            examFormSubmitting = false;
          }, 700);
        }
      });
    }

    if (examsBody) {
      examsBody.addEventListener("click", async function (e) {
        const btn =
          e.target && e.target.closest("button[data-exam-action][data-id]");
        if (!btn) return;
        const examID = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-exam-action") || "");
        if (examID <= 0) return;
        if (action === "use") {
          if (questionExamIDInput) questionExamIDInput.value = String(examID);
          if (resultsExamIDInput) resultsExamIDInput.value = String(examID);
          try {
            await Promise.all([
              loadGuruExamQuestions(examID),
              loadGuruExamAssignments(examID),
            ]);
            setMsg("Exam ID " + examID + " dipilih.");
          } catch (err) {
            setMsg("Gagal memuat data ujian: " + err.message);
          }
          return;
        }
        if (action === "delete") {
          if (!window.confirm("Nonaktifkan ujian ini?")) return;
          const done = beginBusy(btn, msg, "Menonaktifkan ujian...");
          try {
            await api("/api/v1/admin/exams/" + examID, "DELETE");
            await loadGuruExamManageTable();
            setMsg("Ujian berhasil dinonaktifkan.");
          } catch (err) {
            setMsg("Gagal menonaktifkan ujian: " + err.message);
          } finally {
            done();
          }
        }
      });
    }

    if (examQuestionForm) {
      examQuestionForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(examQuestionForm);
        const examID = Number(fd.get("exam_id") || 0);
        if (examID <= 0) {
          setMsg("Isi Exam ID untuk menambah naskah.");
          return;
        }
        const payload = {
          question_id: Number(fd.get("question_id") || 0),
          seq_no: Number(fd.get("seq_no") || 0),
          weight: Number(fd.get("weight") || 1),
        };
        const done = beginBusy(
          examQuestionForm,
          msg,
          "Menyimpan naskah ke ujian...",
        );
        try {
          await api(
            "/api/v1/admin/exams/" + examID + "/questions",
            "POST",
            payload,
          );
          await Promise.all([
            loadGuruExamQuestions(examID),
            loadGuruExamManageTable(),
          ]);
          setMsg("Naskah berhasil dimasukkan ke ujian.");
        } catch (err) {
          setMsg("Gagal menyimpan naskah ke ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examQuestionsLoadBtn) {
      examQuestionsLoadBtn.addEventListener("click", async function () {
        const examID = Number(
          (questionExamIDInput && questionExamIDInput.value) || 0,
        );
        const done = beginBusy(
          examQuestionsLoadBtn,
          msg,
          "Memuat soal ujian...",
        );
        try {
          await loadGuruExamQuestions(examID);
          setMsg("Soal ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat soal ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (examQuestionsBody) {
      examQuestionsBody.addEventListener("click", async function (e) {
        const btn =
          e.target && e.target.closest("button[data-exam-question-del]");
        if (!btn) return;
        const examID = Number(
          (questionExamIDInput && questionExamIDInput.value) || 0,
        );
        const questionID = Number(
          btn.getAttribute("data-exam-question-del") || 0,
        );
        if (examID <= 0 || questionID <= 0) return;
        if (!window.confirm("Hapus soal ini dari ujian?")) return;
        const done = beginBusy(btn, msg, "Menghapus soal dari ujian...");
        try {
          await api(
            "/api/v1/admin/exams/" + examID + "/questions/" + questionID,
            "DELETE",
            {},
          );
          await Promise.all([
            loadGuruExamQuestions(examID),
            loadGuruExamManageTable(),
          ]);
          setMsg("Soal berhasil dihapus dari ujian.");
        } catch (err) {
          setMsg("Gagal menghapus soal dari ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsRefreshBtn) {
      resultsRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(
          resultsRefreshBtn,
          msg,
          "Memuat daftar ujian...",
        );
        try {
          await loadGuruExamManageTable();
          setMsg("Daftar ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat daftar ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsAssignmentForm) {
      resultsAssignmentForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(resultsAssignmentForm);
        const examID = Number(fd.get("exam_id") || 0);
        const done = beginBusy(
          resultsAssignmentForm,
          msg,
          "Memuat peserta ujian...",
        );
        try {
          await loadGuruExamAssignments(examID);
          setMsg("Peserta ujian berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat peserta ujian: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsAttemptForm) {
      resultsAttemptForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(resultsAttemptForm);
        const attemptID = Number(fd.get("attempt_id") || 0);
        if (attemptID <= 0) {
          setMsg("Attempt ID tidak valid.");
          return;
        }
        const done = beginBusy(
          resultsAttemptForm,
          msg,
          "Memuat nilai attempt...",
        );
        try {
          const out = await api(
            "/api/v1/attempts/" + attemptID + "/result",
            "GET",
          );
          latestResultData = out || null;
          const summary = (out && out.summary) || {};
          const itemCount = Array.isArray(out && out.items)
            ? out.items.length
            : 0;
          setResultsSummary(
            "Exam #" +
              String(summary.exam_id || "-") +
              " | Attempt #" +
              String(summary.id || "-") +
              " | Skor: " +
              String(summary.score || 0) +
              " | Benar: " +
              String(summary.total_correct || 0) +
              " | Salah: " +
              String(summary.total_wrong || 0) +
              " | Kosong: " +
              String(summary.total_unanswered || 0) +
              " | Item: " +
              String(itemCount),
          );
          setResultsExtra("Nilai dimuat. Pilih Muat Laporan/Statistik/Grafik.");
          setMsg("Nilai attempt berhasil dimuat.");
        } catch (err) {
          setMsg("Gagal memuat nilai: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (resultsReportBtn) {
      resultsReportBtn.addEventListener("click", function () {
        summarizeAttemptResult("report");
      });
    }
    if (resultsStatsBtn) {
      resultsStatsBtn.addEventListener("click", function () {
        summarizeAttemptResult("stats");
      });
    }
    if (resultsChartBtn) {
      resultsChartBtn.addEventListener("click", function () {
        summarizeAttemptResult("chart");
      });
    }

    if (subjectRefreshBtn) {
      subjectRefreshBtn.addEventListener("click", async function () {
        const done = beginBusy(subjectRefreshBtn, msg, "Memuat ulang mapel...");
        try {
          await Promise.all([loadSubjects(), loadReviewStats()]);
          setMsg("Data mapel berhasil dimuat ulang.");
        } catch (err) {
          setMsg("Gagal memuat mapel: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (subjectAddBtn) {
      subjectAddBtn.hidden = !canManageSubjects;
      if (canManageSubjects) {
        subjectAddBtn.addEventListener("click", function () {
          openSubjectDialog(null);
        });
      }
    }

    if (subjectsBody && canManageSubjects) {
      subjectsBody.addEventListener("click", function (e) {
        const btn =
          e.target && e.target.closest("button[data-action][data-id]");
        if (!btn) return;
        const id = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-action") || "");
        if (!id) return;
        const item = subjectsCache.find(function (it) {
          return Number(it.id) === id;
        });
        if (!item) return;
        if (action === "edit-subject") {
          openSubjectDialog(item);
        } else if (action === "delete-subject") {
          openSubjectDeleteDialog(item);
        }
      });
    }

    if (subjectCancelBtn) {
      subjectCancelBtn.addEventListener("click", function () {
        if (subjectDialog && typeof subjectDialog.close === "function") {
          subjectDialog.close();
        }
      });
    }

    if (subjectDeleteCancelBtn) {
      subjectDeleteCancelBtn.addEventListener("click", function () {
        if (
          subjectDeleteDialog &&
          typeof subjectDeleteDialog.close === "function"
        ) {
          subjectDeleteDialog.close();
        }
      });
    }

    if (subjectDialogForm && canManageSubjects) {
      subjectDialogForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(subjectDialogForm);
        const id = Number(fd.get("id") || 0);
        const payload = {
          education_level: String(fd.get("education_level") || "").trim(),
          subject_type: String(fd.get("subject_type") || "").trim(),
          name: String(fd.get("name") || "").trim(),
        };
        const done = beginBusy(subjectDialogForm, msg, "Menyimpan mapel...");
        try {
          if (id > 0) {
            await api("/api/v1/subjects/" + id, "PUT", payload);
            setMsg("Mapel berhasil diperbarui.");
          } else {
            await api("/api/v1/subjects", "POST", payload);
            setMsg("Mapel berhasil ditambahkan.");
          }
          if (subjectDialog && typeof subjectDialog.close === "function") {
            subjectDialog.close();
          }
          await Promise.all([loadSubjects(), loadReviewStats()]);
        } catch (err) {
          setMsg("Simpan mapel gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (subjectDeleteForm && canManageSubjects) {
      subjectDeleteForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(subjectDeleteForm);
        const id = Number(fd.get("id") || 0);
        if (!id) return;
        const done = beginBusy(subjectDeleteForm, msg, "Menghapus mapel...");
        try {
          await api("/api/v1/subjects/" + id, "DELETE");
          if (
            subjectDeleteDialog &&
            typeof subjectDeleteDialog.close === "function"
          ) {
            subjectDeleteDialog.close();
          }
          await Promise.all([loadSubjects(), loadReviewStats()]);
          setMsg("Mapel berhasil dihapus.");
        } catch (err) {
          setMsg("Hapus mapel gagal: " + err.message);
        } finally {
          done();
        }
      });
    }

    if (reviewDecisionCancelBtn) {
      reviewDecisionCancelBtn.addEventListener("click", function () {
        if (
          reviewDecisionDialog &&
          typeof reviewDecisionDialog.close === "function"
        ) {
          reviewDecisionDialog.close();
        }
      });
    }

    if (reviewBody) {
      reviewBody.addEventListener("click", function (e) {
        const btn =
          e.target && e.target.closest("button[data-action][data-id]");
        if (!btn || !reviewDecisionForm) return;

        const taskID = Number(btn.getAttribute("data-id") || 0);
        const action = String(btn.getAttribute("data-action") || "");
        if (!taskID || (action !== "approve" && action !== "revise")) return;

        const status = action === "approve" ? "disetujui" : "perlu_revisi";
        const title =
          action === "approve" ? "Setujui Tugas Reviu" : "Kirim Perlu Revisi";
        const noteHint =
          action === "approve"
            ? "Catatan opsional untuk persetujuan."
            : "Isi alasan revisi agar penulis mendapat arahan yang jelas.";

        const taskField = reviewDecisionForm.elements.namedItem("task_id");
        const statusField = reviewDecisionForm.elements.namedItem("status");
        if (!taskField || !statusField) return;

        taskField.value = String(taskID);
        statusField.value = status;
        if (reviewDecisionTitle) text(reviewDecisionTitle, title);
        if (reviewDecisionHelp) text(reviewDecisionHelp, noteHint);
        if (reviewDecisionNote) {
          reviewDecisionNote.value = "";
          reviewDecisionNote.required = action === "revise";
        }
        if (
          reviewDecisionDialog &&
          typeof reviewDecisionDialog.showModal === "function"
        ) {
          reviewDecisionDialog.showModal();
        }
      });
    }

    if (reviewDecisionForm) {
      reviewDecisionForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const fd = new FormData(reviewDecisionForm);
        const taskID = Number(fd.get("task_id") || 0);
        const status = String(fd.get("status") || "").trim();
        const note = String(fd.get("note") || "").trim();
        if (!taskID || !status) return;
        if (status === "perlu_revisi" && !note) {
          setMsg("Catatan wajib diisi untuk status perlu_revisi.");
          return;
        }

        const done = beginBusy(
          reviewDecisionForm,
          msg,
          "Menyimpan keputusan...",
        );
        try {
          await api("/api/v1/reviews/tasks/" + taskID + "/decision", "POST", {
            status: status,
            note: note,
          });
          if (
            reviewDecisionDialog &&
            typeof reviewDecisionDialog.close === "function"
          ) {
            reviewDecisionDialog.close();
          }
          const statusEl =
            reviewFilterForm && reviewFilterForm.elements.namedItem("status");
          const statusValue = statusEl ? statusEl.value : "";
          await Promise.all([
            loadReviews(statusValue),
            loadReviewStats(),
            loadSubjects(),
          ]);
          setMsg("Keputusan reviu berhasil disimpan.");
        } catch (err) {
          setMsg("Gagal menyimpan keputusan reviu: " + err.message);
        } finally {
          done();
        }
      });
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
  initHomePage();
  initSimulasiPage();
  initAdminPage();
  initProktorPage();
  initGuruPage();
  initAuthoringPage();
  initAttemptPage();
  initResultPage();
})();
