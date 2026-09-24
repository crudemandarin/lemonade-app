package repository

import (
	"lemonade-api/libraries"
	"lemonade-api/model"
)

type SampleRepository struct {
	DB *libraries.Database
}

func NewSampleRepository(db *libraries.Database) *SampleRepository {
	return &SampleRepository{DB: db}
}

func (r *SampleRepository) FindAll() ([]model.Sample, error) {
	var entries []model.Sample
	if err := r.DB.Find(&entries).Error; err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *SampleRepository) FindByID(id uint) (*model.Sample, error) {
	var entry model.Sample
	if err := r.DB.First(&entry, id).Error; err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *SampleRepository) Create(entry *model.Sample) error {
	return r.DB.Create(entry).Error
}

func (r *SampleRepository) Update(entry *model.Sample) error {
	return r.DB.Save(entry).Error
}

func (r *SampleRepository) Delete(id uint) error {
	return r.DB.Delete(&model.Sample{}, id).Error
}
