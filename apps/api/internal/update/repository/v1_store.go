package repository

import (
	"errors"
	"time"

	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("update: row not found")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ready() bool {
	return s != nil && s.db != nil
}

type Pos struct {
	Created time.Time
	ID      int
}

func afterPos(q *gorm.DB, pos *Pos) *gorm.DB {
	if pos == nil {
		return q
	}
	return q.Where("(created < ? OR (created = ? AND id < ?))", pos.Created, pos.Created, pos.ID)
}

func (s *Store) ListUpdateLogs(pos *Pos, limit int) ([]adminModel.UpdateLog, error) {
	var rows []adminModel.UpdateLog
	err := afterPos(s.db.Model(&adminModel.UpdateLog{}), pos).
		Order("created DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) FindUpdateLog(id int) (*adminModel.UpdateLog, error) {
	var row adminModel.UpdateLog
	err := s.db.Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Store) CreateUpdateLog(row *adminModel.UpdateLog) error {
	return s.db.Create(row).Error
}

func (s *Store) PatchUpdateLog(id int, fields map[string]any) error {
	fields["updated"] = time.Now()
	res := s.db.Model(&adminModel.UpdateLog{}).Where("id = ?", id).UpdateColumns(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteUpdateLog(id int) error {
	res := s.db.Where("id = ?", id).Delete(&adminModel.UpdateLog{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func todosIn(q *gorm.DB, status *int) *gorm.DB {
	if status != nil {
		return q.Where("status = ?", *status)
	}
	return q
}

func (s *Store) ListTodos(status *int, pos *Pos, limit int) ([]adminModel.Todo, error) {
	var rows []adminModel.Todo
	err := afterPos(todosIn(s.db.Model(&adminModel.Todo{}), status), pos).
		Order("created DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) CountTodos(status *int) (int, error) {
	var n int64
	err := todosIn(s.db.Model(&adminModel.Todo{}), status).Count(&n).Error
	return int(n), err
}

func (s *Store) FindTodo(id int) (*adminModel.Todo, error) {
	var row adminModel.Todo
	err := s.db.Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Store) CreateTodo(row *adminModel.Todo) error {
	return s.db.Create(row).Error
}

// PatchTodo edits content only while the row is still in one of the states the
// caller checked, so an edit cannot land on a todo that was completed or
// discarded after it was read.
func (s *Store) PatchTodo(id int, editable []int, fields map[string]any) (moved bool, err error) {
	fields["updated"] = time.Now()
	res := s.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status IN ?", id, editable).
		UpdateColumns(fields)
	return res.RowsAffected > 0, res.Error
}

// TransitionTodo is a guarded UPDATE, not a read-then-write: two users hitting
// 认领 at the same moment both read status 0, both were told they had claimed
// it, and the second write silently replaced the first claimer. moved=false
// means the row left `from` in between.
func (s *Store) TransitionTodo(id, from, to int, fields map[string]any) (moved bool, err error) {
	fields["status"] = to
	fields["updated"] = time.Now()
	res := s.db.Model(&adminModel.Todo{}).
		Where("id = ? AND status = ?", id, from).
		UpdateColumns(fields)
	return res.RowsAffected > 0, res.Error
}

func (s *Store) DeleteTodo(id int) error {
	res := s.db.Where("id = ?", id).Delete(&adminModel.Todo{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
