// Package patient 提供患者服务一期简易实现。
package patient

import (
	"context"

	"yaofang/internal/service/port"
)

// SimplePatientService 一期简易实现：不建患者档案表，返回空集。
// 二期替换为完整患者档案实现，接口不变。
type SimplePatientService struct{}

// NewSimplePatientService 构建简易患者服务。
func NewSimplePatientService() *SimplePatientService { return &SimplePatientService{} }

// Register 一期仅透传返回 0（不落库，患者标识冗余存在处方表中）。
func (s *SimplePatientService) Register(_ context.Context, _ *port.Patient) (int64, error) {
	return 0, nil
}

// GetPatient 一期返回空。
func (s *SimplePatientService) GetPatient(_ context.Context, _ int64) (*port.Patient, error) {
	return nil, nil
}

// GetAllergies 一期返回空集，二期接入处方审核过敏史提醒。
func (s *SimplePatientService) GetAllergies(_ context.Context, _ int64) ([]port.Allergy, error) {
	return nil, nil
}

// GetMedicationHistory 一期返回空集，二期接入重复用药分析。
func (s *SimplePatientService) GetMedicationHistory(_ context.Context, _ int64) ([]port.MedicationRecord, error) {
	return nil, nil
}
