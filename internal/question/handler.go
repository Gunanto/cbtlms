package question

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cbtlms/internal/app/apiresp"
	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc questionService
}

type questionService interface {
	CreateQuestion(ctx context.Context, in CreateQuestionInput) (*QuestionBlueprint, error)
	ListQuestions(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error)
	CreateStimulus(ctx context.Context, in CreateStimulusInput) (*Stimulus, error)
	ListStimuliBySubject(ctx context.Context, subjectID int64) ([]Stimulus, error)
	UpdateStimulus(ctx context.Context, in UpdateStimulusInput) (*Stimulus, error)
	DeleteStimulus(ctx context.Context, stimulusID int64) error
	CreateQuestionVersion(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error)
	UpdateQuestionVersion(ctx context.Context, in UpdateQuestionVersionInput) (*QuestionVersion, error)
	DeleteQuestionVersion(ctx context.Context, questionID int64, versionNo int) error
	FinalizeQuestionVersion(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error)
	RequestReopenFinal(ctx context.Context, in RequestReopenFinalInput) (*ReopenFinalRequest, error)
	ApproveReopenFinal(ctx context.Context, in ApproveReopenFinalInput) (*ReopenFinalRequest, error)
	ListQuestionVersions(ctx context.Context, questionID int64) ([]QuestionVersion, error)
	CreateQuestionParallel(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error)
	ListQuestionParallels(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error)
	UpdateQuestionParallel(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error)
	DeleteQuestionParallel(ctx context.Context, examID, parallelID int64) error
	CreateReviewTask(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error)
	DecideReviewTask(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error)
	ListReviewTasks(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error)
	GetQuestionReviews(ctx context.Context, questionID int64) ([]QuestionReview, error)
}

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type createStimulusRequest struct {
	SubjectID    int64           `json:"subject_id"`
	Title        string          `json:"title"`
	StimulusType string          `json:"stimulus_type"`
	Content      json.RawMessage `json:"content"`
}

type updateStimulusRequest struct {
	SubjectID    int64           `json:"subject_id"`
	Title        string          `json:"title"`
	StimulusType string          `json:"stimulus_type"`
	Content      json.RawMessage `json:"content"`
}

type createQuestionRequest struct {
	SubjectID      int64  `json:"subject_id"`
	QuestionType   string `json:"question_type"`
	Title          string `json:"title"`
	Indicator      string `json:"indicator"`
	Material       string `json:"material"`
	Objective      string `json:"objective"`
	CognitiveLevel string `json:"cognitive_level"`
	Difficulty     *int   `json:"difficulty"`
}

type createQuestionVersionRequest struct {
	StimulusID      *int64                `json:"stimulus_id"`
	StemHTML        *string               `json:"stem_html"`
	ExplanationHTML *string               `json:"explanation_html"`
	HintHTML        *string               `json:"hint_html"`
	AnswerKey       json.RawMessage       `json:"answer_key"`
	Options         []QuestionOptionInput `json:"options"`
	DurationSeconds *int                  `json:"duration_seconds"`
	Weight          *float64              `json:"weight"`
	ChangeNote      *string               `json:"change_note"`
}

type updateQuestionVersionRequest struct {
	StimulusID      *int64                `json:"stimulus_id"`
	StemHTML        *string               `json:"stem_html"`
	ExplanationHTML *string               `json:"explanation_html"`
	HintHTML        *string               `json:"hint_html"`
	AnswerKey       json.RawMessage       `json:"answer_key"`
	Options         []QuestionOptionInput `json:"options"`
	DurationSeconds *int                  `json:"duration_seconds"`
	Weight          *float64              `json:"weight"`
	ChangeNote      *string               `json:"change_note"`
}

type requestReopenFinalRequest struct {
	Reason string `json:"reason"`
}

type approveReopenFinalRequest struct {
	Note string `json:"note"`
}

type createQuestionParallelRequest struct {
	QuestionID    int64  `json:"question_id"`
	ParallelGroup string `json:"parallel_group"`
	ParallelOrder int    `json:"parallel_order"`
	ParallelLabel string `json:"parallel_label"`
}

type createReviewTaskRequest struct {
	QuestionVersionID int64  `json:"question_version_id"`
	ExamID            *int64 `json:"exam_id"`
	ReviewerID        int64  `json:"reviewer_id"`
	Note              string `json:"note"`
}

type reviewDecisionRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type stimulusImportRowError struct {
	Row   int    `json:"row"`
	Title string `json:"title,omitempty"`
	Error string `json:"error"`
}

type stimulusImportReport struct {
	TotalRows   int                      `json:"total_rows"`
	SuccessRows int                      `json:"success_rows"`
	FailedRows  int                      `json:"failed_rows"`
	Errors      []stimulusImportRowError `json:"errors"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateStimulus(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createStimulusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateStimulus(r.Context(), CreateStimulusInput{
		SubjectID:    req.SubjectID,
		Title:        req.Title,
		StimulusType: req.StimulusType,
		Content:      req.Content,
		CreatedBy:    user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrSubjectNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateQuestion(r.Context(), CreateQuestionInput{
		SubjectID:      req.SubjectID,
		QuestionType:   req.QuestionType,
		Title:          req.Title,
		Indicator:      req.Indicator,
		Material:       req.Material,
		Objective:      req.Objective,
		CognitiveLevel: req.CognitiveLevel,
		Difficulty:     req.Difficulty,
		CreatedBy:      user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrSubjectNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var subjectID int64
	subjectIDRaw := strings.TrimSpace(r.URL.Query().Get("subject_id"))
	if subjectIDRaw != "" {
		parsed, err := strconv.ParseInt(subjectIDRaw, 10, 64)
		if err != nil || parsed <= 0 {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "subject_id must be positive"})
			return
		}
		subjectID = parsed
	}

	ownerOnlyRaw := strings.TrimSpace(r.URL.Query().Get("owner_only"))
	ownerOnly := false
	if ownerOnlyRaw != "" {
		parsed, err := strconv.ParseBool(ownerOnlyRaw)
		if err != nil {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "owner_only must be boolean"})
			return
		}
		ownerOnly = parsed
	} else if strings.EqualFold(strings.TrimSpace(user.Role), "guru") {
		// Default mode guru: hanya soal milik sendiri.
		ownerOnly = true
	}

	var ownerID *int64
	if ownerOnly {
		ownerID = &user.ID
	}

	items, err := h.svc.ListQuestions(r.Context(), subjectID, ownerID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) ListStimuli(w http.ResponseWriter, r *http.Request) {
	subjectIDRaw := strings.TrimSpace(r.URL.Query().Get("subject_id"))
	subjectID, err := strconv.ParseInt(subjectIDRaw, 10, 64)
	if err != nil || subjectID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "subject_id is required and must be positive"})
		return
	}

	items, err := h.svc.ListStimuliBySubject(r.Context(), subjectID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) UpdateStimulus(w http.ResponseWriter, r *http.Request) {
	stimulusID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || stimulusID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid stimulus id"})
		return
	}

	var req updateStimulusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateStimulus(r.Context(), UpdateStimulusInput{
		ID:           stimulusID,
		SubjectID:    req.SubjectID,
		Title:        req.Title,
		StimulusType: req.StimulusType,
		Content:      req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrSubjectNotFound), errors.Is(err, ErrStimulusNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteStimulus(w http.ResponseWriter, r *http.Request) {
	stimulusID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || stimulusID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid stimulus id"})
		return
	}
	if err := h.svc.DeleteStimulus(r.Context(), stimulusID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrStimulusNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) ExportStimuliImportTemplateCSV(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("template_import_stimuli_%s.csv", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strings.Join([]string{
		"subject_id,title,stimulus_type,body_html,tabs_json,content_json",
		`1,"Stimulus Single Contoh",single,"<p>Isi konten single berbasis HTML.</p>",,`,
		`1,"Stimulus Multiteks Contoh",multiteks,,"[{""title"":""Tab 1"",""body"":""<p>Konten tab 1</p>""},{""title"":""Tab 2"",""body"":""<p>Konten tab 2</p>""}]",`,
		`1,"Stimulus dengan content_json",single,,,"{""body"":""<p>Alternatif isi via content_json</p>""}"`,
		"",
		"# Catatan:",
		"# 1) Kolom wajib: subject_id,title,stimulus_type.",
		"# 2) stimulus_type: single atau multiteks.",
		"# 3) Untuk single, isi body_html ATAU content_json.",
		"# 4) Untuk multiteks, isi tabs_json ATAU content_json.",
	}, "\n")))
}

func (h *Handler) ImportStimuliCSV(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid multipart form"})
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "file field is required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headerRow, err := reader.Read()
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "gagal membaca header csv"})
		return
	}
	header := map[string]int{}
	for i, raw := range headerRow {
		k := strings.ToLower(strings.TrimSpace(raw))
		if k != "" {
			header[k] = i
		}
	}
	required := []string{"subject_id", "title", "stimulus_type"}
	for _, col := range required {
		if _, ok := header[col]; !ok {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: fmt.Sprintf("kolom wajib tidak ditemukan: %s", col)})
			return
		}
	}

	get := func(row []string, key string) string {
		idx, ok := header[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	report := &stimulusImportReport{Errors: make([]stimulusImportRowError, 0)}
	rowNo := 1
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		rowNo++
		if readErr != nil {
			report.TotalRows++
			report.FailedRows++
			report.Errors = append(report.Errors, stimulusImportRowError{
				Row:   rowNo,
				Error: "format csv tidak valid",
			})
			continue
		}
		if isCSVRowEmpty(record) {
			continue
		}

		report.TotalRows++

		subjectIDRaw := get(record, "subject_id")
		title := get(record, "title")
		stimulusType := strings.ToLower(get(record, "stimulus_type"))
		bodyHTML := get(record, "body_html")
		tabsJSON := get(record, "tabs_json")
		contentJSON := get(record, "content_json")

		subjectID, convErr := strconv.ParseInt(subjectIDRaw, 10, 64)
		if convErr != nil || subjectID <= 0 {
			report.FailedRows++
			report.Errors = append(report.Errors, stimulusImportRowError{
				Row:   rowNo,
				Title: title,
				Error: "subject_id harus angka positif",
			})
			continue
		}

		var content json.RawMessage
		if strings.TrimSpace(contentJSON) != "" {
			var parsed any
			if err := json.Unmarshal([]byte(contentJSON), &parsed); err != nil {
				report.FailedRows++
				report.Errors = append(report.Errors, stimulusImportRowError{
					Row:   rowNo,
					Title: title,
					Error: "content_json bukan JSON valid",
				})
				continue
			}
			content = json.RawMessage(contentJSON)
		} else {
			switch stimulusType {
			case "single":
				payload, _ := json.Marshal(map[string]any{
					"body": bodyHTML,
				})
				content = payload
			case "multiteks":
				if strings.TrimSpace(tabsJSON) == "" {
					report.FailedRows++
					report.Errors = append(report.Errors, stimulusImportRowError{
						Row:   rowNo,
						Title: title,
						Error: "tabs_json wajib diisi untuk stimulus multiteks",
					})
					continue
				}
				var tabs any
				if err := json.Unmarshal([]byte(tabsJSON), &tabs); err != nil {
					report.FailedRows++
					report.Errors = append(report.Errors, stimulusImportRowError{
						Row:   rowNo,
						Title: title,
						Error: "tabs_json bukan JSON valid",
					})
					continue
				}
				payload, _ := json.Marshal(map[string]any{
					"tabs": tabs,
				})
				content = payload
			default:
				// Delegasikan pesan validasi final ke service.
				content = json.RawMessage(`{}`)
			}
		}

		_, createErr := h.svc.CreateStimulus(r.Context(), CreateStimulusInput{
			SubjectID:    subjectID,
			Title:        title,
			StimulusType: stimulusType,
			Content:      content,
			CreatedBy:    user.ID,
		})
		if createErr != nil {
			report.FailedRows++
			report.Errors = append(report.Errors, stimulusImportRowError{
				Row:   rowNo,
				Title: title,
				Error: createErr.Error(),
			})
			continue
		}
		report.SuccessRows++
	}

	if report.TotalRows == 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "tidak ada baris data pada file csv"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{
		"filename": hdr.Filename,
		"report":   report,
		"summary":  "import completed",
	}})
}

func isCSVRowEmpty(row []string) bool {
	for _, col := range row {
		if strings.TrimSpace(col) != "" {
			return false
		}
	}
	return true
}

func (h *Handler) CreateQuestionVersion(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	var req createQuestionVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateQuestionVersion(r.Context(), CreateQuestionVersionInput{
		QuestionID:      questionID,
		StimulusID:      req.StimulusID,
		StemHTML:        req.StemHTML,
		ExplanationHTML: req.ExplanationHTML,
		HintHTML:        req.HintHTML,
		AnswerKey:       req.AnswerKey,
		Options:         req.Options,
		DurationSeconds: req.DurationSeconds,
		Weight:          req.Weight,
		ChangeNote:      req.ChangeNote,
		CreatedBy:       user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) FinalizeQuestionVersion(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	item, err := h.svc.FinalizeQuestionVersion(r.Context(), questionID, versionNo)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) RequestReopenFinal(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	var req requestReopenFinalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	out, err := h.svc.RequestReopenFinal(r.Context(), RequestReopenFinalInput{
		QuestionID:    questionID,
		VersionNo:     versionNo,
		Reason:        req.Reason,
		RequestedBy:   user.ID,
		RequestedRole: user.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrReopenPending), errors.Is(err, ErrReopenNotAllowed):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: out})
}

func (h *Handler) ApproveReopenFinal(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	var req approveReopenFinalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	out, err := h.svc.ApproveReopenFinal(r.Context(), ApproveReopenFinalInput{
		QuestionID:   questionID,
		VersionNo:    versionNo,
		ApproverID:   user.ID,
		ApproverRole: user.Role,
		Note:         req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: out})
}

func (h *Handler) UpdateQuestionVersion(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	var req updateQuestionVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateQuestionVersion(r.Context(), UpdateQuestionVersionInput{
		QuestionID:      questionID,
		VersionNo:       versionNo,
		StimulusID:      req.StimulusID,
		StemHTML:        req.StemHTML,
		ExplanationHTML: req.ExplanationHTML,
		HintHTML:        req.HintHTML,
		AnswerKey:       req.AnswerKey,
		Options:         req.Options,
		DurationSeconds: req.DurationSeconds,
		Weight:          req.Weight,
		ChangeNote:      req.ChangeNote,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotFound), errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteQuestionVersion(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	if err := h.svc.DeleteQuestionVersion(r.Context(), questionID, versionNo); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{"deleted": true}})
}

func (h *Handler) ListQuestionVersions(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	items, err := h.svc.ListQuestionVersions(r.Context(), questionID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) CreateQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}

	var req createQuestionParallelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateQuestionParallel(r.Context(), CreateQuestionParallelInput{
		ExamID:        examID,
		QuestionID:    req.QuestionID,
		ParallelGroup: req.ParallelGroup,
		ParallelOrder: req.ParallelOrder,
		ParallelLabel: req.ParallelLabel,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotInExam):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListQuestionParallels(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelGroup := strings.TrimSpace(r.URL.Query().Get("parallel_group"))

	items, err := h.svc.ListQuestionParallels(r.Context(), examID, parallelGroup)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) UpdateQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelID, err := strconv.ParseInt(chi.URLParam(r, "parallelID"), 10, 64)
	if err != nil || parallelID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid parallel id"})
		return
	}

	var req createQuestionParallelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateQuestionParallel(r.Context(), UpdateQuestionParallelInput{
		ExamID:        examID,
		ParallelID:    parallelID,
		QuestionID:    req.QuestionID,
		ParallelGroup: req.ParallelGroup,
		ParallelOrder: req.ParallelOrder,
		ParallelLabel: req.ParallelLabel,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrParallelNotFound), errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotInExam):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelID, err := strconv.ParseInt(chi.URLParam(r, "parallelID"), 10, 64)
	if err != nil || parallelID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid parallel id"})
		return
	}

	if err := h.svc.DeleteQuestionParallel(r.Context(), examID, parallelID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrParallelNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) CreateReviewTask(w http.ResponseWriter, r *http.Request) {
	var req createReviewTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateReviewTask(r.Context(), CreateReviewTaskInput{
		QuestionVersionID: req.QuestionVersionID,
		ExamID:            req.ExamID,
		ReviewerID:        req.ReviewerID,
		Note:              req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound), errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) DecideReviewTask(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid review task id"})
		return
	}

	var req reviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.DecideReviewTask(r.Context(), ReviewDecisionInput{
		TaskID:       taskID,
		ReviewerID:   user.ID,
		IsPrivileged: user.Role == "admin" || user.Role == "proktor",
		Status:       req.Status,
		Note:         req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewTaskNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListReviewTasks(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	reviewerID := user.ID
	if user.Role == "admin" || user.Role == "proktor" {
		if raw := strings.TrimSpace(r.URL.Query().Get("reviewer_id")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
				reviewerID = parsed
			}
		}
	}

	items, err := h.svc.ListReviewTasks(r.Context(), reviewerID, status)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) GetQuestionReviews(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	items, err := h.svc.GetQuestionReviews(r.Context(), questionID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func writeJSON(w http.ResponseWriter, r *http.Request, code int, payload apiResponse) {
	if payload.OK {
		apiresp.WriteOK(w, r, code, payload.Data)
		return
	}
	apiresp.WriteError(w, r, code, payload.Error)
}
