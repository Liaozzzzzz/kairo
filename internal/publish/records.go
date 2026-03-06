package publish

import "Kairo/internal/db/schema"

func (p *PublishManager) ListRecords(taskID string) ([]schema.PublishRecord, error) {
	return p.publishRecordDAL.ListRecords(p.ctx, taskID)
}

func (p *PublishManager) updateRecord(record *schema.PublishRecord, status schema.PublishStatus, message string) *schema.PublishRecord {
	record.Status = status
	record.Message = message
	return record
}
