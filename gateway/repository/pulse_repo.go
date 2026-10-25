package repository

import (
	"errors"
	"gateway/entity"
	"time"

	"gorm.io/gorm"
)

var ErrPulseNotFound = errors.New("pulse not found")

type PulseRepository struct {
	db *gorm.DB
}

func NewPulseRepository(db *gorm.DB) *PulseRepository {
	return &PulseRepository{db: db}
}

func (r *PulseRepository) List(userID uint) ([]entity.Pulse, error) {
	var pulses []entity.Pulse
	err := r.db.Where("user_id = ?", userID).Order("created_at ASC, id ASC").Find(&pulses).Error
	return pulses, err
}

func (r *PulseRepository) Owned(userID, id uint) (*entity.Pulse, error) {
	var pulse entity.Pulse
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&pulse).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPulseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pulse, nil
}

// CreateLegacy keeps the statement that was saved before pulses existed.
// Its ledger key is the user id, which is where those rows already live.
func (r *PulseRepository) CreateLegacy(userID uint) (*entity.Pulse, error) {
	pulse := &entity.Pulse{
		UserID:    userID,
		Name:      "Выписка",
		LedgerKey: int64(userID),
	}
	if err := r.db.Create(pulse).Error; err != nil {
		if isDuplicate(err) {
			var existing entity.Pulse
			err = r.db.Where("user_id = ? AND ledger_key = ?", userID, int64(userID)).First(&existing).Error
			if err != nil {
				return nil, err
			}
			return &existing, nil
		}
		return nil, err
	}
	return pulse, nil
}

// Create stores a new pulse. The key is assigned after the row has an id,
// so two pulses of the same account never share a statement.
func (r *PulseRepository) Create(userID uint, name string) (*entity.Pulse, error) {
	pulse := &entity.Pulse{
		UserID:    userID,
		Name:      name,
		LedgerKey: time.Now().UnixNano(),
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pulse).Error; err != nil {
			return err
		}
		pulse.LedgerKey = 1_000_000_000 + int64(pulse.ID)
		return tx.Model(pulse).Update("ledger_key", pulse.LedgerKey).Error
	})
	if err != nil {
		return nil, err
	}
	return pulse, nil
}

func (r *PulseRepository) Rename(userID, id uint, name string) error {
	res := r.db.Model(&entity.Pulse{}).Where("id = ? AND user_id = ?", id, userID).Update("name", name)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPulseNotFound
	}
	return nil
}

func (r *PulseRepository) Delete(userID, id uint) error {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&entity.Pulse{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPulseNotFound
	}
	return nil
}
