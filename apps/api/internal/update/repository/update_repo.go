package repository

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type UpdateRepository struct {
	db *gorm.DB
}

func NewUpdateRepository(db *gorm.DB) *UpdateRepository {
	return &UpdateRepository{db: db}
}

func (r *UpdateRepository) CountHistory() int64 {
	var total int64
	r.db.Model(&adminModel.UpdateLog{}).Count(&total)
	return total
}

func (r *UpdateRepository) FindHistoryPaginated(page, limit int) []adminModel.UpdateLog {
	var logs []adminModel.UpdateLog
	r.db.Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&logs)
	return logs
}

func (r *UpdateRepository) CreateHistory(log *adminModel.UpdateLog) error {
	return r.db.Create(log).Error
}

func (r *UpdateRepository) UpdateHistory(id int, fields map[string]any) error {
	return r.db.Model(&adminModel.UpdateLog{}).Where("id = ?", id).
		Updates(fields).Error
}

func (r *UpdateRepository) DeleteHistory(id int) error {
	return r.db.Delete(&adminModel.UpdateLog{}, id).Error
}

func (r *UpdateRepository) todos(status *int) *gorm.DB {
	q := r.db.Model(&adminModel.Todo{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	return q
}

func (r *UpdateRepository) CountTodos(status *int) int64 {
	var total int64
	r.todos(status).Count(&total)
	return total
}

func (r *UpdateRepository) FindTodosPaginated(page, limit int, status *int) []adminModel.Todo {
	var todos []adminModel.Todo
	r.todos(status).Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&todos)
	return todos
}

func (r *UpdateRepository) CreateTodo(todo *adminModel.Todo) error {
	return r.db.Create(todo).Error
}

func (r *UpdateRepository) FindTodoByID(id int) (*adminModel.Todo, error) {
	var todo adminModel.Todo
	err := r.db.First(&todo, id).Error
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

// The transitions below are guarded UPDATEs, not read-then-writes: two
// users hitting 认领 at the same moment both read status 0, both were told
// they had claimed it, and the second write silently replaced the first
// claimer. moved=false means the row left the expected state in between.

func (r *UpdateRepository) ClaimTodo(id, userID int) (moved bool, err error) {
	res := r.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, adminModel.TodoStatusPending).
		Updates(map[string]any{"status": adminModel.TodoStatusClaimed, "claimed_user_id": userID})
	return res.RowsAffected > 0, res.Error
}

// ReleaseTodo is ClaimTodo's inverse. Clearing claimed_user_id is the point:
// leaving it set would keep the row answering "已被 X 认领" while it sits in
// the pending filter, and the same user could not claim it again.
func (r *UpdateRepository) ReleaseTodo(id int) (moved bool, err error) {
	res := r.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, adminModel.TodoStatusClaimed).
		Updates(map[string]any{
			"status":          adminModel.TodoStatusPending,
			"claimed_user_id": nil,
			"completed_time":  nil,
		})
	return res.RowsAffected > 0, res.Error
}

// CompleteTodo carries no actor. An earlier attempt to let editors close a
// claimed todo also kept `claimed_user_id = <requester>` here, so every such
// UPDATE matched zero rows and the caller answered a passing permission check
// with "该待办的状态刚刚发生了变化, 请刷新后重试". The status CAS is the whole
// concurrency guard: a claimed todo cannot change claimer without leaving
// status 1 first, and who may end it is the handler's decision.
func (r *UpdateRepository) CompleteTodo(id int) (moved bool, err error) {
	res := r.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, adminModel.TodoStatusClaimed).
		Updates(map[string]any{"status": adminModel.TodoStatusDone, "completed_time": time.Now()})
	return res.RowsAffected > 0, res.Error
}

func (r *UpdateRepository) DiscardTodo(id, fromStatus int) (moved bool, err error) {
	res := r.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(map[string]any{"status": adminModel.TodoStatusDiscarded, "completed_time": nil})
	return res.RowsAffected > 0, res.Error
}

// ReopenTodo is DiscardTodo's inverse, restricted by the caller to the
// admin/ren `update_log.reopen` permission. Clearing claimed_user_id keeps the
// reopened row honest: a todo discarded from 进行中 still carries its old
// claimer, and leaving it would show "已被 X 认领" on a pending row.
func (r *UpdateRepository) ReopenTodo(id int) (moved bool, err error) {
	res := r.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, adminModel.TodoStatusDiscarded).
		Updates(map[string]any{
			"status":          adminModel.TodoStatusPending,
			"claimed_user_id": nil,
			"completed_time":  nil,
		})
	return res.RowsAffected > 0, res.Error
}

func (r *UpdateRepository) UpdateTodo(id int, fields map[string]any) error {
	return r.db.Model(&adminModel.Todo{}).Where("id = ?", id).
		Updates(fields).Error
}

func (r *UpdateRepository) DeleteTodo(id int) error {
	return r.db.Delete(&adminModel.Todo{}, id).Error
}
