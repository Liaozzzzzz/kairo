package publish

import (
	"Kairo/internal/db/schema"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 获取发布任务列表
func (p *PublishManager) ListTasks(statusFilter, platformFilter string, page, pageSize int) (*schema.PublishTaskListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	tasks, total, err := p.publishTaskDAL.ListTasks(p.ctx, statusFilter, platformFilter, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &schema.PublishTaskListResponse{
		Total: total,
		Data:  tasks,
	}, nil
}

// 获取单个发布任务
func (p *PublishManager) GetTaskById(taskID string) (*schema.PublishTask, error) {
	return p.publishTaskDAL.GetTaskById(p.ctx, taskID)
}

// 手动创建发布任务
func (p *PublishManager) CreateTask(req schema.CreatePublishTaskRequest) (*schema.PublishTask, error) {
	if strings.TrimSpace(req.AccountID) == "" {
		log.Printf("[CreateTask] account id is required")
		return nil, fmt.Errorf("account id is required")
	}
	account, err := p.publishAccountDAL.GetAccountById(p.ctx, req.AccountID)
	if err != nil {
		log.Printf("[CreateTask] account not found: %v", err)
		return nil, fmt.Errorf("account not found: %v", err)
	}
	if !account.IsEnabled {
		log.Printf("[CreateTask] account is disabled: %v", err)
		return nil, fmt.Errorf("account is disabled")
	}
	if strings.TrimSpace(account.PlatformID) == "" {
		log.Printf("[CreateTask] account platform not configured")
		return nil, fmt.Errorf("account platform not configured")
	}
	if strings.TrimSpace(account.CookiePath) == "" {
		log.Printf("[CreateTask] account cookie is empty")
		return nil, fmt.Errorf("account cookie is empty")
	}

	if _, statErr := os.Stat(account.CookiePath); statErr != nil {
		log.Printf("[CreateTask] account cookie not found: %v", statErr)
		return nil, fmt.Errorf("account cookie not found")
	}
	if strings.TrimSpace(req.HighlightID) == "" {
		log.Printf("[CreateTask] highlight id is required")
		return nil, fmt.Errorf("highlight id is required")
	}

	highlight, err := p.videoDAL.GetHighlightByID(p.ctx, req.HighlightID)
	if err != nil {
		log.Printf("[CreateTask] highlight not found: %v", err)
		return nil, fmt.Errorf("highlight not found: %v", err)
	}
	if strings.TrimSpace(highlight.FilePath) == "" {
		log.Printf("[CreateTask] highlight file is empty")
		return nil, fmt.Errorf("highlight file is empty")
	}

	// Check if a similar publish task already exists for this highlight and account
	exists, err := p.publishTaskDAL.ExistsByHighlightAndAccount(p.ctx, req.HighlightID, req.AccountID)
	if err != nil {
		log.Printf("[CreateTask] failed to check existing tasks: %v", err)
		return nil, fmt.Errorf("failed to check existing tasks: %v", err)
	}
	if exists {
		log.Printf("[CreateTask] similar publish task already exists for highlight %s and account %s", req.HighlightID, req.AccountID)
		return nil, fmt.Errorf("similar publish task already exists")
	}

	publishType := req.PublishType
	if publishType == "" {
		publishType = schema.PublishTypeManual
	}

	// 创建发布任务
	task := &schema.PublishTask{
		ID:          uuid.New().String(),
		HighlightID: req.HighlightID,
		AccountID:   req.AccountID,
		Status:      schema.PublishStatusPending,
		Type:        publishType,
		ScheduledAt: req.ScheduledAt,
		Title:       req.Title,
		Description: req.Description,
		Tags:        strings.TrimSpace(req.Tags),
	}

	if err := p.publishTaskDAL.SaveTask(p.ctx, task); err != nil {
		log.Printf("[CreateTask] failed to create publish task: %v", err)
		return nil, fmt.Errorf("failed to create publish task: %v", err)
	}

	return task, nil
}

func (p *PublishManager) UpdateTask(req schema.UpdatePublishTaskRequest) (*schema.PublishTask, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, fmt.Errorf("task id is required")
	}
	task, err := p.publishTaskDAL.GetTaskById(p.ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %v", err)
	}
	if task.Status != schema.PublishStatusPending && task.Status != schema.PublishStatusFailed {
		return nil, fmt.Errorf("can only update pending or failed task")
	}

	task.ScheduledAt = req.ScheduledAt
	task.Title = req.Title
	task.Description = req.Description
	task.Tags = strings.TrimSpace(req.Tags)
	task.ErrorMessage = ""
	if req.ScheduledAt > 0 {
		task.Type = schema.PublishTypeAuto
	} else {
		task.Type = schema.PublishTypeManual
	}
	if err := p.publishTaskDAL.SaveTask(p.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update publish task: %v", err)
	}
	return task, nil
}

// 执行发布任务
func (p *PublishManager) publishTask(task *schema.PublishTask, trigger string) error {
	record := &schema.PublishRecord{
		ID:      uuid.New().String(),
		TaskID:  task.ID,
		Trigger: trigger,
		Status:  schema.PublishStatusPending,
	}
	_ = p.publishRecordDAL.SaveRecord(p.ctx, record)

	// 获取平台信息
	if strings.TrimSpace(task.AccountID) == "" {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Account not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Account not found")
	}
	account, err := p.publishAccountDAL.GetAccountById(p.ctx, task.AccountID)
	if err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Account not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Account not found")
	}
	if !account.IsEnabled {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Account disabled"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Account disabled")
	}
	platform, err := p.publishPlatformDAL.GetPlatformById(p.ctx, account.PlatformID)
	if err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Platform not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Platform not found")
	}

	// 创建平台API实例
	platformAPI, err := p.platformManager.CreatePlatformAPI(*platform)
	if err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, fmt.Sprintf("Failed to create platform API: %v", err)))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, fmt.Sprintf("Failed to create platform API: %v", err))
	}

	// 获取视频文件路径
	if strings.TrimSpace(task.HighlightID) == "" {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Highlight not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Highlight not found")
	}
	highlight, err := p.videoDAL.GetHighlightByID(p.ctx, task.HighlightID)
	if err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Highlight not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Highlight not found")
	}
	videoPath := strings.TrimSpace(highlight.FilePath)

	// 验证文件存在
	if _, err := filepath.Abs(videoPath); err != nil || videoPath == "" {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Video file not accessible"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Video file not accessible")
	}
	if _, err := os.Stat(videoPath); err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Video file not found"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Video file not found")
	}

	accountCookiePath := ""
	if task.AccountID != "" {
		if strings.TrimSpace(account.CookiePath) == "" {
			_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Account cookie missing"))
			return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Account cookie missing")
		}
		accountCookiePath = account.CookiePath
	}

	// 验证Cookie文件存在
	if _, err := os.Stat(accountCookiePath); err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, "Account cookie not accessible"))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, "Account cookie not accessible")
	}

	// 解析标签
	var tags []string
	if task.Tags != "" {
		tags = strings.Split(task.Tags, ",")
	}

	var scheduledAt *time.Time
	if task.ScheduledAt > 0 {
		t := time.UnixMilli(task.ScheduledAt)
		scheduledAt = &t
	}
	platformVideoID, err := platformAPI.UploadVideo(p.ctx, task.Title, task.Description, tags, videoPath, accountCookiePath, scheduledAt)
	if err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, fmt.Sprintf("Upload failed: %v", err)))
		return p.updateTaskStatus(task.ID, schema.PublishStatusFailed, fmt.Sprintf("Upload failed: %v", err))
	}

	// 更新任务状态
	task.PlatformVideoID = platformVideoID
	task.Status = schema.PublishStatusPublished
	task.PublishedAt = time.Now().UnixMilli()

	if err := p.publishTaskDAL.SaveTask(p.ctx, task); err != nil {
		_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusFailed, fmt.Sprintf("Failed to save task: %v", err)))
		return fmt.Errorf("failed to update task status: %v", err)
	}

	_ = p.publishRecordDAL.SaveRecord(p.ctx, p.updateRecord(record, schema.PublishStatusPublished, ""))
	return nil
}

// 更新任务状态
func (p *PublishManager) updateTaskStatus(taskID string, status schema.PublishStatus, errorMessage string) error {
	return p.publishTaskDAL.UpdateTaskStatus(p.ctx, taskID, status, errorMessage)
}

// 取消发布任务
func (p *PublishManager) CancelTask(taskID string) error {
	task, err := p.publishTaskDAL.GetTaskById(p.ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %v", err)
	}

	if task.Status == schema.PublishStatusPublished {
		return fmt.Errorf("cannot cancel already published task")
	}

	return p.updateTaskStatus(taskID, schema.PublishStatusCancelled, "Cancelled by user")
}

// 重新发布失败的任务
func (p *PublishManager) RetryTask(taskID string) error {
	task, err := p.publishTaskDAL.GetTaskById(p.ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %v", err)
	}

	if task.Status != schema.PublishStatusFailed {
		return fmt.Errorf("can only retry failed tasks")
	}

	// 重置任务状态
	task.Status = schema.PublishStatusPending
	task.ErrorMessage = ""
	task.PublishedAt = 0

	return p.publishTaskDAL.SaveTask(p.ctx, task)
}

func (p *PublishManager) PublishTaskNow(taskID string) error {
	task, err := p.publishTaskDAL.GetTaskById(p.ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %v", err)
	}
	if task.Status == schema.PublishStatusPublished {
		return fmt.Errorf("cannot publish already published task")
	}
	if task.Status == schema.PublishStatusCancelled {
		return fmt.Errorf("cannot publish cancelled task")
	}
	task.Status = schema.PublishStatusPending
	task.ErrorMessage = ""
	task.ScheduledAt = time.Now().UnixMilli()
	if err := p.publishTaskDAL.SaveTask(p.ctx, task); err != nil {
		return err
	}
	return p.publishTask(task, "manual")
}

// 删除发布任务
func (p *PublishManager) DeleteTask(taskID string) error {
	task, err := p.publishTaskDAL.GetTaskById(p.ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %v", err)
	}

	if task.Status != schema.PublishStatusCancelled && task.Status != schema.PublishStatusFailed {
		return fmt.Errorf("can only delete cancelled or failed tasks")
	}

	if err := p.publishRecordDAL.DeleteRecordsByTaskID(p.ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task records: %v", err)
	}

	return p.publishTaskDAL.DeleteTask(p.ctx, taskID)
}
