package handler

import (
	"net/http"

	appmiddleware "bluecollarjob/internal/middleware"
	"bluecollarjob/internal/models"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type EmployerHandler struct {
	employers service.EmployerService
}

func NewEmployerHandler(employers service.EmployerService) *EmployerHandler {
	return &EmployerHandler{employers: employers}
}

type registerEmployerRequest struct {
	CompanyName string  `json:"company_name"`
	ContactName string  `json:"contact_name"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	PhoneNumber *string `json:"phone_number"`
	City        *string `json:"city"`
	State       *string `json:"state"`
}

type loginEmployerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateEmployerProfileRequest struct {
	CompanyName string  `json:"company_name"`
	ContactName string  `json:"contact_name"`
	PhoneNumber *string `json:"phone_number"`
	City        *string `json:"city"`
	State       *string `json:"state"`
}

type employerJobRequest struct {
	Title                    string                  `json:"title"`
	Role                     string                  `json:"role"`
	Description              string                  `json:"description"`
	SkillCategory            string                  `json:"skill_category"`
	LocationCity             string                  `json:"location_city"`
	LocationState            string                  `json:"location_state"`
	ShiftSchedule            string                  `json:"shift_schedule"`
	WageMinPaise             *int                    `json:"wage_min_paise"`
	WageMaxPaise             *int                    `json:"wage_max_paise"`
	RequiredVerificationTier models.VerificationTier `json:"required_verification_tier"`
	Openings                 int                     `json:"openings"`
	IsActive                 *bool                   `json:"is_active"`
}

type updateJobStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

func (h *EmployerHandler) Register(c *gin.Context) {
	var request registerEmployerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if !validateEmployerRegistration(c, request.CompanyName, request.Email, request.Password) {
		return
	}

	contactName := requiredString(request.ContactName)
	if contactName == "" {
		contactName = requiredString(request.CompanyName)
	}

	employer, token, err := h.employers.Register(c.Request.Context(), service.RegisterEmployerInput{
		CompanyName: requiredString(request.CompanyName),
		ContactName: contactName,
		Email:       requiredString(request.Email),
		Password:    request.Password,
		PhoneNumber: optionalString(request.PhoneNumber),
		City:        optionalString(request.City),
		State:       optionalString(request.State),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"employer": newEmployerResponse(employer), "token": token})
}

func (h *EmployerHandler) Login(c *gin.Context) {
	var request loginEmployerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	request.Email = requiredString(request.Email)
	if request.Email == "" || !emailPattern.MatchString(request.Email) {
		writeError(c, http.StatusBadRequest, "invalid_email", "valid email is required")
		return
	}
	if request.Password == "" {
		writeError(c, http.StatusBadRequest, "missing_password", "password is required")
		return
	}

	employer, token, err := h.employers.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employer": newEmployerResponse(employer), "token": token})
}

func (h *EmployerHandler) GetMe(c *gin.Context) {
	employer, err := h.employers.GetProfile(c.Request.Context(), appmiddleware.GetEmployerID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employer": newEmployerResponse(employer)})
}

func (h *EmployerHandler) UpdateMe(c *gin.Context) {
	var request updateEmployerProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if requiredString(request.CompanyName) == "" {
		writeError(c, http.StatusBadRequest, "missing_company_name", "company_name is required")
		return
	}
	contactName := requiredString(request.ContactName)
	if contactName == "" {
		contactName = requiredString(request.CompanyName)
	}

	employer, err := h.employers.UpdateProfile(c.Request.Context(), appmiddleware.GetEmployerID(c), service.UpdateEmployerProfileInput{
		CompanyName: requiredString(request.CompanyName),
		ContactName: contactName,
		PhoneNumber: optionalString(request.PhoneNumber),
		City:        optionalString(request.City),
		State:       optionalString(request.State),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employer": newEmployerResponse(employer)})
}

func (h *EmployerHandler) CreateJob(c *gin.Context) {
	input, ok := bindEmployerJobRequest(c)
	if !ok {
		return
	}
	job, err := h.employers.CreateJob(c.Request.Context(), appmiddleware.GetEmployerID(c), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"job": newJobResponse(*job)})
}

func (h *EmployerHandler) ListJobs(c *gin.Context) {
	limit, offset := parsePagination(c)
	jobs, err := h.employers.ListJobs(c.Request.Context(), appmiddleware.GetEmployerID(c), limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response := make([]jobResponse, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, newJobResponse(job))
	}
	c.JSON(http.StatusOK, gin.H{"jobs": response})
}

func (h *EmployerHandler) GetJob(c *gin.Context) {
	job, err := h.employers.GetJob(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": newJobResponse(*job)})
}

func (h *EmployerHandler) UpdateJob(c *gin.Context) {
	input, ok := bindEmployerJobRequest(c)
	if !ok {
		return
	}
	job, err := h.employers.UpdateJob(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": newJobResponse(*job)})
}

func (h *EmployerHandler) UpdateJobStatus(c *gin.Context) {
	var request updateJobStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if request.IsActive == nil {
		writeError(c, http.StatusBadRequest, "missing_is_active", "is_active is required")
		return
	}
	job, err := h.employers.UpdateJobStatus(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"), *request.IsActive)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": newJobResponse(*job)})
}

func validateEmployerRegistration(c *gin.Context, companyName, email, password string) bool {
	if requiredString(companyName) == "" {
		writeError(c, http.StatusBadRequest, "missing_company_name", "company_name is required")
		return false
	}
	if requiredString(email) == "" || !emailPattern.MatchString(requiredString(email)) {
		writeError(c, http.StatusBadRequest, "invalid_email", "valid email is required")
		return false
	}
	if password == "" {
		writeError(c, http.StatusBadRequest, "missing_password", "password is required")
		return false
	}
	return true
}

func bindEmployerJobRequest(c *gin.Context) (service.EmployerJobInput, bool) {
	var request employerJobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return service.EmployerJobInput{}, false
	}
	if requiredString(request.Title) == "" {
		writeError(c, http.StatusBadRequest, "missing_job_title", "job title is required")
		return service.EmployerJobInput{}, false
	}
	if requiredString(request.Role) == "" {
		writeError(c, http.StatusBadRequest, "missing_job_role", "job role is required")
		return service.EmployerJobInput{}, false
	}
	if requiredString(request.LocationCity) == "" || requiredString(request.LocationState) == "" {
		writeError(c, http.StatusBadRequest, "missing_job_location", "job location is required")
		return service.EmployerJobInput{}, false
	}
	if requiredString(request.ShiftSchedule) == "" {
		writeError(c, http.StatusBadRequest, "missing_shift_schedule", "shift_schedule is required")
		return service.EmployerJobInput{}, false
	}
	if request.Openings <= 0 {
		request.Openings = 1
	}
	if request.SkillCategory == "" {
		request.SkillCategory = request.Role
	}
	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}
	if request.Description == "" {
		request.Description = request.Title
	}
	return service.EmployerJobInput{
		Title:                    requiredString(request.Title),
		Role:                     requiredString(request.Role),
		Description:              requiredString(request.Description),
		SkillCategory:            requiredString(request.SkillCategory),
		LocationCity:             requiredString(request.LocationCity),
		LocationState:            requiredString(request.LocationState),
		ShiftSchedule:            requiredString(request.ShiftSchedule),
		WageMinPaise:             request.WageMinPaise,
		WageMaxPaise:             request.WageMaxPaise,
		RequiredVerificationTier: request.RequiredVerificationTier,
		Openings:                 request.Openings,
		IsActive:                 isActive,
	}, true
}
