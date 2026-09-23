package repository

import (
	"errors"
	"math"
	"strings"
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("toolset: row not found")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ready() bool {
	return s != nil && s.db != nil
}
func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func UniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

type ListFilter struct {
	Type      string
	Language  string
	Platform  string
	Version   string
	Q         string
	UserID    int
	AuthorIDs []int
}

type SortSpec struct {
	Column string
	Desc   bool
}

func (f ListFilter) apply(q *gorm.DB) *gorm.DB {
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Language != "" {
		q = q.Where("language = ?", f.Language)
	}
	if f.Platform != "" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.Version != "" {
		q = q.Where("version = ?", f.Version)
	}
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if kw := strings.TrimSpace(f.Q); kw != "" {
		q = q.Where(`name ILIKE ? ESCAPE '\'`, "%"+escapeLike(kw)+"%")
	}
	if f.AuthorIDs != nil {
		q = q.Where("user_id IN ?", f.AuthorIDs)
	}
	return q
}

func (s *Store) DistinctAuthors(f ListFilter) ([]int, error) {
	f.AuthorIDs = nil
	var ids []int
	err := f.apply(s.db.Model(&model.GalgameToolset{})).Distinct("user_id").Pluck("user_id", &ids).Error
	return ids, err
}

func (s *Store) Count(f ListFilter) (int, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := f.apply(s.db.Model(&model.GalgameToolset{})).Count(&n).Error
	return int(n), err
}

func (s *Store) List(f ListFilter, sort SortSpec, offset, limit int) ([]model.GalgameToolset, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return nil, nil
	}
	dir := "ASC"
	if sort.Desc {
		dir = "DESC"
	}
	var rows []model.GalgameToolset
	err := f.apply(s.db.Model(&model.GalgameToolset{})).
		Order(sort.Column + " " + dir + ", id " + dir).
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *Store) Find(id int) (*model.GalgameToolset, error) {
	var row model.GalgameToolset
	err := s.db.First(&row, id).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) IncrementView(id int) error {
	res := s.db.Exec(`UPDATE galgame_toolset SET view = view + 1 WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) IncrementDownload(id int) error {
	res := s.db.Exec(`UPDATE galgame_toolset_resource SET download = download + 1 WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Aliases(toolsetIDs []int) (map[int][]string, error) {
	out := map[int][]string{}
	if len(toolsetIDs) == 0 {
		return out, nil
	}
	var rows []model.GalgameToolsetAlias
	if err := s.db.Where("toolset_id IN ?", toolsetIDs).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ToolsetID] = append(out[r.ToolsetID], r.Name)
	}
	return out, nil
}

type PracticalityAgg struct {
	Average      *float64
	Count        int
	Distribution [5]int
}

func (s *Store) Practicality(toolsetIDs []int) (map[int]PracticalityAgg, error) {
	out := map[int]PracticalityAgg{}
	if len(toolsetIDs) == 0 {
		return out, nil
	}
	type row struct {
		ToolsetID int
		Rate      int
		N         int
	}
	var rows []row
	err := s.db.Model(&model.GalgameToolsetPracticality{}).
		Select("toolset_id, rate, COUNT(*) AS n").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id, rate").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	sums := map[int]int{}
	for _, r := range rows {
		agg := out[r.ToolsetID]
		agg.Count += r.N
		if r.Rate >= 1 && r.Rate <= 5 {
			agg.Distribution[r.Rate-1] = r.N
		}
		sums[r.ToolsetID] += r.Rate * r.N
		out[r.ToolsetID] = agg
	}
	for id, agg := range out {
		if agg.Count > 0 {
			rounded := math.Round(float64(sums[id])/float64(agg.Count)*100) / 100
			agg.Average = &rounded
			out[id] = agg
		}
	}
	return out, nil
}

func (s *Store) UserRating(toolsetID, userID int) (*int, error) {
	var p model.GalgameToolsetPracticality
	err := s.db.Where("toolset_id = ? AND user_id = ?", toolsetID, userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rate := p.Rate
	return &rate, nil
}

func (s *Store) UpsertPracticality(toolsetID, userID, rate int) error {
	return s.db.Exec(`
		INSERT INTO galgame_toolset_practicality (rate, user_id, toolset_id, created, updated)
		VALUES (?, ?, ?, now(), now())
		ON CONFLICT (toolset_id, user_id) DO UPDATE SET rate = EXCLUDED.rate, updated = now()`,
		rate, userID, toolsetID).Error
}

func (s *Store) DownloadSums(toolsetIDs []int) (map[int]int, error) {
	out := map[int]int{}
	if len(toolsetIDs) == 0 {
		return out, nil
	}
	type row struct {
		ToolsetID int
		Total     int
	}
	var rows []row
	err := s.db.Model(&model.GalgameToolsetResource{}).
		Select("toolset_id, COALESCE(SUM(download), 0) AS total").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ToolsetID] = r.Total
	}
	return out, nil
}

func (s *Store) ContributorIDs(toolsetID int) ([]int, error) {
	var ids []int
	err := s.db.Model(&model.GalgameToolsetContributor{}).
		Where("toolset_id = ?", toolsetID).
		Order("id").
		Pluck("user_id", &ids).Error
	return ids, err
}

func (s *Store) Resources(toolsetID int) ([]model.GalgameToolsetResource, error) {
	var rows []model.GalgameToolsetResource
	err := s.db.Where("toolset_id = ?", toolsetID).
		Order("created DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

func (s *Store) FindResource(id int) (*model.GalgameToolsetResource, error) {
	var row model.GalgameToolsetResource
	err := s.db.First(&row, id).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) UploadsByUUIDs(uuids []string) (map[string]model.ToolsetUpload, error) {
	out := map[string]model.ToolsetUpload{}
	if len(uuids) == 0 {
		return out, nil
	}
	var rows []model.ToolsetUpload
	if err := s.db.Where("artifact_uuid IN ?", uuids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ArtifactUUID] = r
	}
	return out, nil
}

func (s *Store) CreateToolset(row *model.GalgameToolset, aliases []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		if err := replaceAliases(tx, row.ID, aliases); err != nil {
			return err
		}
		return addContributor(tx, row.ID, row.UserID)
	})
}

func (s *Store) PatchToolset(id int, fields map[string]any, aliases []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			res := tx.Model(&model.GalgameToolset{}).Where("id = ?", id).UpdateColumns(fields)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrNotFound
			}
		}
		if aliases != nil {
			return replaceAliases(tx, id, aliases)
		}
		return nil
	})
}

func (s *Store) DeleteToolset(id int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		related := []any{
			&model.GalgameToolsetAlias{},
			&model.GalgameToolsetContributor{},
			&model.GalgameToolsetPracticality{},
			&model.GalgameToolsetResource{},
			&model.GalgameToolsetCategoryRelation{},
			&model.ToolsetUpload{},
		}
		for _, m := range related {
			if err := tx.Where("toolset_id = ?", id).Delete(m).Error; err != nil {
				return err
			}
		}
		res := tx.Delete(&model.GalgameToolset{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func replaceAliases(tx *gorm.DB, toolsetID int, aliases []string) error {
	if err := tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetAlias{}).Error; err != nil {
		return err
	}
	for _, name := range aliases {
		if err := tx.Create(&model.GalgameToolsetAlias{Name: name, ToolsetID: toolsetID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func addContributor(tx *gorm.DB, toolsetID, userID int) error {
	return tx.Exec(`
		INSERT INTO galgame_toolset_contributor (toolset_id, user_id, created, updated)
		VALUES (?, ?, now(), now())
		ON CONFLICT (toolset_id, user_id) DO NOTHING`, toolsetID, userID).Error
}
func (s *Store) CreateResource(row *model.GalgameToolsetResource) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		if err := addContributor(tx, row.ToolsetID, row.UserID); err != nil {
			return err
		}
		return tx.Model(&model.GalgameToolset{}).Where("id = ?", row.ToolsetID).
			Update("resource_update_time", time.Now()).Error
	})
}

func (s *Store) PatchResource(id int, fields map[string]any) error {
	res := s.db.Model(&model.GalgameToolsetResource{}).Where("id = ?", id).UpdateColumns(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteResource(id int) error {
	res := s.db.Delete(&model.GalgameToolsetResource{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) FileResources(toolsetID int) ([]model.GalgameToolsetResource, error) {
	var rows []model.GalgameToolsetResource
	err := s.db.Where("toolset_id = ? AND type = ?", toolsetID, "s3").Find(&rows).Error
	return rows, err
}

func (s *Store) InsertUpload(row *model.ToolsetUpload) error {
	return s.db.Create(row).Error
}

func (s *Store) FindUpload(toolsetID int, uuid string) (*model.ToolsetUpload, error) {
	var row model.ToolsetUpload
	err := s.db.Where("artifact_uuid = ? AND toolset_id = ?", uuid, toolsetID).First(&row).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) DeleteUpload(uuid string) error {
	res := s.db.Where("artifact_uuid = ?", uuid).Delete(&model.ToolsetUpload{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CompleteUpload(uuid string, userID int, fileSize int64) (first bool, err error) {
	err = s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`
			UPDATE toolset_upload SET completed_at = now()
			WHERE artifact_uuid = ? AND completed_at IS NULL`, uuid)
		if res.Error != nil {
			return res.Error
		}
		first = res.RowsAffected == 1
		if !first {
			return nil
		}
		return tx.Model(&userModel.KungalUserState{}).Where("user_id = ?", userID).
			Updates(map[string]any{
				"daily_toolset_upload_count": gorm.Expr("daily_toolset_upload_count + 1"),
				"daily_toolset_upload_bytes": gorm.Expr("daily_toolset_upload_bytes + ?", fileSize),
			}).Error
	})
	return first, err
}

func (s *Store) DailyUploadState(userID int) (bytes int64, moe int, err error) {
	var state userModel.KungalUserState
	err = s.db.Select("daily_toolset_upload_bytes, moemoepoint").
		Where("user_id = ?", userID).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return state.DailyToolsetUploadBytes, state.Moemoepoint, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	).Replace(s)
}
