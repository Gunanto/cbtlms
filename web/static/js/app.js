(function () {
  function cookieValue(name) {
    const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
    return match ? decodeURIComponent(match[1]) : '';
  }

  function csrfHeaders(method) {
    const headers = { 'Content-Type': 'application/json' };
    const m = String(method || 'GET').toUpperCase();
    if (m === 'POST' || m === 'PUT' || m === 'PATCH' || m === 'DELETE') {
      const csrf = cookieValue('cbtlms_csrf');
      if (csrf) headers['X-CSRF-Token'] = csrf;
    }
    return headers;
  }

  async function api(path, method, body) {
    const res = await fetch(path, {
      method,
      credentials: 'include',
      headers: csrfHeaders(method),
      body: body ? JSON.stringify(body) : undefined,
    });
    const json = await res.json().catch(function () { return {}; });
    if (!res.ok || !json.ok) {
      const msg = json && json.error && json.error.message ? json.error.message : 'Request failed';
      throw new Error(msg);
    }
    return json.data;
  }

  async function meOrNull() {
    try {
      return await api('/api/v1/auth/me', 'GET');
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
    return String(input || '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
  }

  async function initLoginPage() {
    const form = document.getElementById('login-form');
    if (!form) return;
    const msg = document.getElementById('login-message');

    const user = await meOrNull();
    if (user) {
      window.location.href = '/simulasi';
      return;
    }

    form.addEventListener('submit', async function (e) {
      e.preventDefault();
      text(msg, 'Memproses login...');
      const fd = new FormData(form);
      try {
        await api('/api/v1/auth/login-password', 'POST', {
          identifier: String(fd.get('identifier') || ''),
          password: String(fd.get('password') || ''),
        });
        window.location.href = '/simulasi';
      } catch (err) {
        text(msg, 'Login gagal: ' + err.message);
      }
    });
  }

  async function initSimulasiPage() {
    const form = document.getElementById('simulasi-form');
    if (!form) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = '/login';
      return;
    }

    const msg = document.getElementById('simulasi-message');
    const levelSelect = document.getElementById('level-select');
    const typeSelect = document.getElementById('type-select');
    const subjectSelect = document.getElementById('subject-select');
    const examSelect = document.getElementById('exam-select');
    const startBtn = document.getElementById('start-btn');

    const subjects = await api('/api/v1/subjects', 'GET');
    const levels = [...new Set(subjects.map(function (s) { return s.education_level; }))];

    levels.forEach(function (lvl) {
      const o = document.createElement('option');
      o.value = lvl;
      o.textContent = lvl;
      levelSelect.appendChild(o);
    });

    function resetSelect(sel, placeholder) {
      sel.innerHTML = '';
      const o = document.createElement('option');
      o.value = '';
      o.textContent = placeholder;
      sel.appendChild(o);
      sel.disabled = true;
    }

    levelSelect.addEventListener('change', function () {
      resetSelect(typeSelect, 'Pilih jenis...');
      resetSelect(subjectSelect, 'Pilih mapel...');
      resetSelect(examSelect, 'Pilih exam...');
      startBtn.disabled = true;

      const level = levelSelect.value;
      if (!level) return;

      const types = [...new Set(subjects
        .filter(function (s) { return s.education_level === level; })
        .map(function (s) { return s.subject_type; }))];

      typeSelect.disabled = false;
      types.forEach(function (t) {
        const o = document.createElement('option');
        o.value = t;
        o.textContent = t;
        typeSelect.appendChild(o);
      });
    });

    typeSelect.addEventListener('change', function () {
      resetSelect(subjectSelect, 'Pilih mapel...');
      resetSelect(examSelect, 'Pilih exam...');
      startBtn.disabled = true;

      const level = levelSelect.value;
      const type = typeSelect.value;
      if (!level || !type) return;

      subjectSelect.disabled = false;
      subjects
        .filter(function (s) { return s.education_level === level && s.subject_type === type; })
        .forEach(function (s) {
          const o = document.createElement('option');
          o.value = String(s.id);
          o.textContent = s.name;
          subjectSelect.appendChild(o);
        });
    });

    subjectSelect.addEventListener('change', async function () {
      resetSelect(examSelect, 'Pilih exam...');
      startBtn.disabled = true;

      const subjectID = subjectSelect.value;
      if (!subjectID) return;

      try {
        const exams = await api('/api/v1/exams?subject_id=' + encodeURIComponent(subjectID), 'GET');
        examSelect.disabled = false;
        exams.forEach(function (x) {
          const o = document.createElement('option');
          o.value = String(x.id);
          o.textContent = x.code + ' - ' + x.title;
          examSelect.appendChild(o);
        });
      } catch (err) {
        text(msg, 'Gagal load exam: ' + err.message);
      }
    });

    examSelect.addEventListener('change', function () {
      startBtn.disabled = !examSelect.value;
    });

    form.addEventListener('submit', async function (e) {
      e.preventDefault();
      const examID = Number(examSelect.value || 0);
      if (!examID) return;

      try {
        const payload = { exam_id: examID };
        if (user.role === 'admin' || user.role === 'proktor') {
          payload.student_id = user.id;
        }
        const out = await api('/api/v1/attempts/start', 'POST', payload);
        window.location.href = '/ujian/' + out.id;
      } catch (err) {
        text(msg, 'Gagal start attempt: ' + err.message);
      }
    });
  }

  function readSelectedFromPayload(payload) {
    if (!payload || typeof payload !== 'object') return [];
    const v = payload.selected;
    if (typeof v === 'string') return v ? [v] : [];
    if (Array.isArray(v)) return v.filter(function (x) { return typeof x === 'string' && x.trim() !== ''; });
    return [];
  }

  async function initAttemptPage() {
    const root = document.getElementById('attempt-root');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = '/login';
      return;
    }

    const attemptID = Number(root.getAttribute('data-attempt-id') || 0);
    let currentNo = 1;
    let totalQuestions = 1;
    let currentQuestion = null;
    let remainInterval = null;

    const remainingLabel = document.getElementById('remaining-label');
    const message = document.getElementById('attempt-message');
    const questionTitle = document.getElementById('question-title');
    const stimulusBox = document.getElementById('stimulus-box');
    const stemBox = document.getElementById('stem-box');
    const optionsBox = document.getElementById('options-box');
    const doubtCheckbox = document.getElementById('doubt-checkbox');

    function formatRemain(secs) {
      const s = Math.max(0, Number(secs || 0));
      const h = Math.floor(s / 3600);
      const m = Math.floor((s % 3600) / 60);
      const ss = s % 60;
      return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0') + ':' + String(ss).padStart(2, '0');
    }

    async function loadSummary() {
      const sum = await api('/api/v1/attempts/' + attemptID, 'GET');
      totalQuestions = Number(sum.total_questions || 1);
      let remain = Number(sum.remaining_secs || 0);
      text(remainingLabel, formatRemain(remain));
      if (remainInterval) clearInterval(remainInterval);
      remainInterval = setInterval(function () {
        remain = Math.max(0, remain - 1);
        text(remainingLabel, formatRemain(remain));
      }, 1000);
      return sum;
    }

    function renderOptions(q) {
      let payload = {};
      try { payload = JSON.parse(q.answer_payload || '{}'); } catch (_) {}

      const selected = new Set(readSelectedFromPayload(payload));
      const qType = String(q.question_type || '').toLowerCase();

      if (qType === 'benar_salah_pernyataan') {
        let key = {};
        try { key = JSON.parse(q.answer_key || '{}'); } catch (_) {}
        const statements = Array.isArray(key.statements) ? key.statements : [];
        if (!statements.length) {
          html(optionsBox, '<p class="muted">Pernyataan tidak tersedia.</p>');
          return;
        }

        const existing = {};
        if (Array.isArray(payload.answers)) {
          payload.answers.forEach(function (a) {
            if (a && typeof a.id === 'string' && typeof a.value === 'boolean') {
              existing[a.id] = a.value;
            }
          });
        }

        const rows = statements.map(function (s, idx) {
          const id = String(s.id || ('s' + idx));
          const yesChecked = existing[id] === true ? 'checked' : '';
          const noChecked = existing[id] === false ? 'checked' : '';
          return [
            '<div class="option-row">',
            '<div><strong>' + escapeHtml(id) + '</strong></div>',
            '<label><input type="radio" name="bs_' + escapeHtml(id) + '" value="true" ' + yesChecked + '> Benar</label>',
            '<label><input type="radio" name="bs_' + escapeHtml(id) + '" value="false" ' + noChecked + '> Salah</label>',
            '</div>'
          ].join(' ');
        });
        html(optionsBox, rows.join(''));
        return;
      }

      const multiple = qType === 'multi_jawaban';
      const inputType = multiple ? 'checkbox' : 'radio';
      const rows = (q.options || []).map(function (opt) {
        const checked = selected.has(opt.option_key) ? 'checked' : '';
        return [
          '<label class="option-row">',
          '<input type="' + inputType + '" name="option" value="' + escapeHtml(opt.option_key) + '" ' + checked + '>',
          '<div><strong>(' + escapeHtml(opt.option_key) + ')</strong> ' + opt.option_html + '</div>',
          '</label>'
        ].join('');
      });
      html(optionsBox, rows.join(''));
    }

    function collectPayload(q) {
      const qType = String(q.question_type || '').toLowerCase();
      if (qType === 'benar_salah_pernyataan') {
        let key = {};
        try { key = JSON.parse(q.answer_key || '{}'); } catch (_) {}
        const statements = Array.isArray(key.statements) ? key.statements : [];
        const answers = [];

        statements.forEach(function (s, idx) {
          const id = String(s.id || ('s' + idx));
          const selected = document.querySelector('input[name="bs_' + CSS.escape(id) + '"]:checked');
          if (selected && (selected.value === 'true' || selected.value === 'false')) {
            answers.push({ id: id, value: selected.value === 'true' });
          }
        });
        return { answers: answers };
      }

      const selectedNodes = optionsBox.querySelectorAll('input[name="option"]:checked');
      const selected = Array.from(selectedNodes).map(function (n) { return String(n.value || '').trim(); }).filter(Boolean);
      if (qType === 'pg_tunggal') {
        return { selected: selected[0] || '' };
      }
      return { selected: selected };
    }

    async function saveCurrent() {
      if (!currentQuestion) return;
      await api('/api/v1/attempts/' + attemptID + '/answers/' + currentQuestion.question_id, 'PUT', {
        answer_payload: collectPayload(currentQuestion),
        is_doubt: !!doubtCheckbox.checked,
      });
    }

    async function loadQuestion(no) {
      const q = await api('/api/v1/attempts/' + attemptID + '/questions/' + no, 'GET');
      currentQuestion = q;
      currentNo = no;
      text(questionTitle, 'Soal nomor ' + no + ' dari ' + totalQuestions);
      html(stemBox, q.stem_html || '<p class="muted">Soal tidak tersedia.</p>');
      html(stimulusBox, q.stimulus_html || '<p class="muted">Tidak ada stimulus.</p>');
      doubtCheckbox.checked = !!q.is_doubt;
      renderOptions(q);
      text(message, '');
    }

    document.getElementById('prev-btn').addEventListener('click', async function () {
      if (currentNo <= 1) return;
      try { await saveCurrent(); await loadQuestion(currentNo - 1); }
      catch (err) { text(message, 'Gagal pindah soal: ' + err.message); }
    });

    document.getElementById('next-btn').addEventListener('click', async function () {
      if (currentNo >= totalQuestions) return;
      try { await saveCurrent(); await loadQuestion(currentNo + 1); }
      catch (err) { text(message, 'Gagal pindah soal: ' + err.message); }
    });

    document.getElementById('submit-btn').addEventListener('click', async function () {
      if (!window.confirm('Yakin submit final?')) return;
      try {
        await saveCurrent();
        await api('/api/v1/attempts/' + attemptID + '/submit', 'POST');
        window.location.href = '/hasil/' + attemptID;
      } catch (err) {
        text(message, 'Gagal submit: ' + err.message);
      }
    });

    optionsBox.addEventListener('change', async function () {
      try { await saveCurrent(); text(message, 'Tersimpan otomatis.'); }
      catch (err) { text(message, 'Autosave gagal: ' + err.message); }
    });

    doubtCheckbox.addEventListener('change', async function () {
      try { await saveCurrent(); text(message, 'Status ragu-ragu tersimpan.'); }
      catch (err) { text(message, 'Simpan ragu-ragu gagal: ' + err.message); }
    });

    try { await loadSummary(); await loadQuestion(1); }
    catch (err) { text(message, 'Gagal memuat attempt: ' + err.message); }
  }

  async function initResultPage() {
    const root = document.getElementById('result-root');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = '/login';
      return;
    }

    const attemptID = Number(root.getAttribute('data-attempt-id') || 0);
    const summaryBox = document.getElementById('result-summary');
    const tbody = document.querySelector('#result-table tbody');

    try {
      const out = await api('/api/v1/attempts/' + attemptID + '/result', 'GET');
      const s = out.summary;
      html(summaryBox,
        '<strong>Status:</strong> ' + escapeHtml(s.status) +
        ' | <strong>Skor:</strong> ' + escapeHtml(String(s.score)) +
        ' | <strong>Benar:</strong> ' + escapeHtml(String(s.total_correct)) +
        ' | <strong>Salah:</strong> ' + escapeHtml(String(s.total_wrong)) +
        ' | <strong>Kosong:</strong> ' + escapeHtml(String(s.total_unanswered))
      );

      tbody.innerHTML = '';
      out.items.forEach(function (it, idx) {
        const tr = document.createElement('tr');
        const breakdown = Array.isArray(it.breakdown) && it.breakdown.length
          ? '<br><small>' + it.breakdown.map(function (b) { return escapeHtml(b.id + ':' + (b.correct ? 'ok' : 'x')); }).join(', ') + '</small>'
          : '';
        tr.innerHTML = [
          '<td>' + (idx + 1) + '</td>',
          '<td>' + escapeHtml(String(it.question_id)) + '</td>',
          '<td>' + escapeHtml((it.selected || []).join(', ')) + '</td>',
          '<td>' + escapeHtml((it.correct || []).join(', ')) + '</td>',
          '<td>' + escapeHtml(it.reason || '') + breakdown + '</td>',
          '<td>' + escapeHtml(String(it.earned_score || 0)) + '</td>'
        ].join('');
        tbody.appendChild(tr);
      });
    } catch (err) {
      text(summaryBox, 'Gagal memuat hasil: ' + err.message);
    }
  }

  async function initAuthoringPage() {
    const root = document.querySelector('[data-page="authoring_content"]');
    if (!root) return;

    const user = await meOrNull();
    if (!user) {
      window.location.href = '/login';
      return;
    }
    if (!['admin', 'proktor', 'guru'].includes(String(user.role || ''))) {
      alert('Halaman authoring hanya untuk admin/proktor/guru');
      window.location.href = '/';
      return;
    }

    const message = document.getElementById('authoring-message');
    const stimulusOut = document.getElementById('stimulus-output');
    const versionOut = document.getElementById('version-output');
    const parallelOut = document.getElementById('parallel-output');

    function pretty(el, data) {
      if (el) el.textContent = JSON.stringify(data, null, 2);
    }
    function setMsg(msg) {
      text(message, msg);
    }

    const stimulusForm = document.getElementById('stimulus-form');
    if (stimulusForm) {
      stimulusForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusForm);
        try {
          const out = await api('/api/v1/stimuli', 'POST', {
            subject_id: Number(fd.get('subject_id') || 0),
            title: String(fd.get('title') || ''),
            stimulus_type: String(fd.get('stimulus_type') || ''),
            content: JSON.parse(String(fd.get('content_raw') || '{}')),
          });
          pretty(stimulusOut, out);
          setMsg('Stimulus berhasil disimpan.');
        } catch (err) {
          setMsg('Gagal simpan stimulus: ' + err.message);
        }
      });
    }

    const stimulusListForm = document.getElementById('stimulus-list-form');
    if (stimulusListForm) {
      stimulusListForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(stimulusListForm);
        try {
          const out = await api('/api/v1/stimuli?subject_id=' + encodeURIComponent(String(fd.get('subject_id') || '')), 'GET');
          pretty(stimulusOut, out);
          setMsg('List stimulus berhasil dimuat.');
        } catch (err) {
          setMsg('Gagal list stimulus: ' + err.message);
        }
      });
    }

    const versionCreateForm = document.getElementById('version-create-form');
    if (versionCreateForm) {
      versionCreateForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(versionCreateForm);
        try {
          const qid = Number(fd.get('question_id') || 0);
          const payload = {
            stem_html: String(fd.get('stem_html') || ''),
            explanation_html: String(fd.get('explanation_html') || ''),
            hint_html: String(fd.get('hint_html') || ''),
            answer_key: JSON.parse(String(fd.get('answer_key_raw') || '{}')),
          };
          const stimulusID = Number(fd.get('stimulus_id') || 0);
          if (stimulusID > 0) payload.stimulus_id = stimulusID;
          const durationRaw = String(fd.get('duration_seconds') || '').trim();
          if (durationRaw !== '') payload.duration_seconds = Number(durationRaw);
          const weightRaw = String(fd.get('weight') || '').trim();
          if (weightRaw !== '') payload.weight = Number(weightRaw);
          const changeNote = String(fd.get('change_note') || '').trim();
          if (changeNote) payload.change_note = changeNote;

          const out = await api('/api/v1/questions/' + qid + '/versions', 'POST', payload);
          pretty(versionOut, out);
          setMsg('Draft versi soal berhasil disimpan.');
        } catch (err) {
          setMsg('Gagal simpan versi: ' + err.message);
        }
      });
    }

    const versionFinalizeForm = document.getElementById('version-finalize-form');
    if (versionFinalizeForm) {
      versionFinalizeForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(versionFinalizeForm);
        try {
          const qid = Number(fd.get('question_id') || 0);
          const ver = Number(fd.get('version_no') || 0);
          const out = await api('/api/v1/questions/' + qid + '/versions/' + ver + '/finalize', 'POST');
          pretty(versionOut, out);
          setMsg('Versi berhasil difinalkan.');
        } catch (err) {
          setMsg('Gagal finalize versi: ' + err.message);
        }
      });
    }

    const versionListForm = document.getElementById('version-list-form');
    if (versionListForm) {
      versionListForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(versionListForm);
        try {
          const qid = Number(fd.get('question_id') || 0);
          const out = await api('/api/v1/questions/' + qid + '/versions', 'GET');
          pretty(versionOut, out);
          setMsg('List versi berhasil dimuat.');
        } catch (err) {
          setMsg('Gagal list versi: ' + err.message);
        }
      });
    }

    const parallelCreateForm = document.getElementById('parallel-create-form');
    if (parallelCreateForm) {
      parallelCreateForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(parallelCreateForm);
        try {
          const examID = Number(fd.get('exam_id') || 0);
          const out = await api('/api/v1/exams/' + examID + '/parallels', 'POST', {
            question_id: Number(fd.get('question_id') || 0),
            parallel_group: String(fd.get('parallel_group') || 'default'),
            parallel_order: Number(fd.get('parallel_order') || 0),
            parallel_label: String(fd.get('parallel_label') || ''),
          });
          pretty(parallelOut, out);
          setMsg('Parallel berhasil ditambahkan.');
        } catch (err) {
          setMsg('Gagal tambah parallel: ' + err.message);
        }
      });
    }

    const parallelListForm = document.getElementById('parallel-list-form');
    if (parallelListForm) {
      parallelListForm.addEventListener('submit', async function (e) {
        e.preventDefault();
        const fd = new FormData(parallelListForm);
        try {
          const examID = Number(fd.get('exam_id') || 0);
          const group = String(fd.get('parallel_group') || '').trim();
          const qs = group ? ('?parallel_group=' + encodeURIComponent(group)) : '';
          const out = await api('/api/v1/exams/' + examID + '/parallels' + qs, 'GET');
          pretty(parallelOut, out);
          setMsg('List parallel berhasil dimuat.');
        } catch (err) {
          setMsg('Gagal list parallel: ' + err.message);
        }
      });
    }
  }

  initLoginPage();
  initSimulasiPage();
  initAuthoringPage();
  initAttemptPage();
  initResultPage();
})();
