package service

import (
	"errors"

	"lemonade-api/model"
	"lemonade-api/repository"
)

var ErrSampleNameRequired = errors.New("name is required")

type SampleService struct {
	Repository *repository.SampleRepository
}

func NewSampleService(repository *repository.SampleRepository) *SampleService {
	return &SampleService{Repository: repository}
}

func (s *SampleService) List() ([]model.Sample, error) {
	return s.Repository.FindAll()
}

func (s *SampleService) Get(id uint) (*model.Sample, error) {
	return s.Repository.FindByID(id)
}

func (s *SampleService) Create(name string) (*model.Sample, error) {
	if name == "" {
		return nil, ErrSampleNameRequired
	}

	sample := &model.Sample{Name: name}
	if err := s.Repository.Create(sample); err != nil {
		return nil, err
	}

	return sample, nil
}

func (s *SampleService) Update(id uint, name string) (*model.Sample, error) {
	if name == "" {
		return nil, ErrSampleNameRequired
	}

	sample, err := s.Repository.FindByID(id)
	if err != nil {
		return nil, err
	}

	sample.Name = name
	if err := s.Repository.Update(sample); err != nil {
		return nil, err
	}

	return sample, nil
}

func (s *SampleService) Delete(id uint) error {
	return s.Repository.Delete(id)
}
