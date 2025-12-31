package core

import (
	"fmt"
	"sync"
)

// StateUpdater interface'i, Engine'in state paketine doğrudan bağımlı olmamasını sağlar.
type StateUpdater interface {
	UpdateResource(resType, name, targetState, status string) error
}

// ConfigItem, motorun işleyeceği ham konfigürasyon parçasıdır.
type ConfigItem struct {
	Name   string
	Type   string
	State  string
	Params map[string]interface{}
}

// Engine, kaynakları yöneten ana yapıdır.
type Engine struct {
	Context        *SystemContext
	StateUpdater   StateUpdater // Opsiyonel: State yöneticisi
	AppliedHistory []ApplyableResource
}

// NewEngine yeni bir motor örneği oluşturur.
func NewEngine(ctx *SystemContext, updater StateUpdater) *Engine {
	// Backup Yöneticisini başlat
	_ = InitBackupManager() // Hata olursa şimdilik yoksay (veya logla)
	return &Engine{
		Context:      ctx,
		StateUpdater: updater,
	}
}

// ResourceCreator fonksiyon tipi
type ResourceCreator func(resType, name string, params map[string]interface{}, ctx *SystemContext) (ApplyableResource, error)

// ApplyableResource arayüzü
type ApplyableResource interface {
	Apply(ctx *SystemContext) (Result, error)
	GetName() string
}

// Run, verilen konfigürasyon listesini işler.
func (e *Engine) Run(items []ConfigItem, createFn ResourceCreator) error {
	errCount := 0

	for _, item := range items {
		// Params hazırlığı
		if item.Params == nil {
			item.Params = make(map[string]interface{})
		}
		item.Params["state"] = item.State

		// 1. Kaynağı oluştur
		res, err := createFn(item.Type, item.Name, item.Params, e.Context)
		if err != nil {
			Failure(err, "Skipping invalid resource definition: "+item.Name)
			errCount++
			continue
		}

		// 2. Kaynağı uygula
		result, err := res.Apply(e.Context)

		status := "success"
		if err != nil {
			status = "failed"
			errCount++
			fmt.Printf("❌ [%s] Failed: %v\n", item.Name, err)
		} else if result.Changed {
			fmt.Printf("✅ [%s] %s\n", item.Name, result.Message)
		} else {
			fmt.Printf("ℹ️  [%s] OK\n", item.Name)
		}

		// 3. Durumu Kaydet (Eğer DryRun değilse)
		if !e.Context.DryRun && e.StateUpdater != nil {
			// Başarısız olsa bile son deneme durumunu "failed" olarak kaydediyoruz
			saveErr := e.StateUpdater.UpdateResource(item.Type, item.Name, item.State, status)
			if saveErr != nil {
				fmt.Printf("⚠️ Warning: Failed to save state for %s: %v\n", item.Name, saveErr)
			}
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors during execution", errCount)
	}
	return nil
}

// RunParallel, verilen layer'daki konfigürasyon parçalarını paralel işler.
func (e *Engine) RunParallel(layer []ConfigItem, createFn ResourceCreator) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(layer))
	var updatedResources []ApplyableResource // Başarılı olanları takip et (Rollback için)
	var mu sync.Mutex                        // updatedResources için lock

	for _, item := range layer {
		wg.Add(1)
		go func(it ConfigItem) {
			defer wg.Done()

			// Params hazırlığı
			if it.Params == nil {
				it.Params = make(map[string]interface{})
			}
			it.Params["state"] = it.State

			// 1. Kaynağı oluştur
			res, err := createFn(it.Type, it.Name, it.Params, e.Context)
			if err != nil {
				Failure(err, "Skipping invalid resource definition: "+it.Name)
				errChan <- err
				return
			}

			// 2. Kaynağı uygula
			result, err := res.Apply(e.Context)

			status := "success"
			if err != nil {
				status = "failed"
				errChan <- err
				fmt.Printf("❌ [%s] Failed: %v\n", it.Name, err)
			} else if result.Changed {
				fmt.Printf("✅ [%s] %s\n", it.Name, result.Message)

				// Başarılı değişiklikleri kaydet (Rollback için)
				if !e.Context.DryRun {
					mu.Lock()
					updatedResources = append(updatedResources, res)
					mu.Unlock()
				}
			} else {
				fmt.Printf("ℹ️  [%s] OK\n", it.Name)
			}

			// 3. Durumu Kaydet
			if !e.Context.DryRun && e.StateUpdater != nil {
				e.StateUpdater.UpdateResource(it.Type, it.Name, it.State, status)
			}
		}(item)
	}

	wg.Wait()
	close(errChan)

	// Hata var mı kontrol et
	errCount := 0
	for range errChan {
		errCount++
	}

	if errCount > 0 {
		// Rollback Tetikle
		if !e.Context.DryRun {
			fmt.Printf("\n🚨 Error occurred. Initiating Rollback...\n")

			// 1. Önce şu anki katmanda başarılı olmuş (ancak diğerlerinin hatası yüzünden yarım kalmış) işlemleri geri al
			fmt.Printf("Visualizing Rollback for current layer (%d resources)...\n", len(updatedResources))
			e.rollback(updatedResources)

			// 2. Önceki katmanlarda tamamlanmış işlemleri geri al
			fmt.Printf("Visualizing Rollback for previous layers (%d resources)...\n", len(e.AppliedHistory))
			e.rollback(e.AppliedHistory)
		}
		return fmt.Errorf("encountered %d errors in parallel layer execution", errCount)
	}

	// Başarılı olanları global geçmişe ekle
	// Not: Revert sırası için LIFO olması gerekir. rollback fonksiyonu listeyi tersten geziyor.
	// AppliedHistory'ye eklerken FIFO ekliyoruz (append).
	// Örnek: Layer0 (A, B) -> AppliedHistory=[A, B]
	// Layer1 (C, D) -> Fail. CurrentRevert(C). HistoryRevert(A, B) -> B revert, A revert. Correct.
	e.AppliedHistory = append(e.AppliedHistory, updatedResources...)

	return nil
}

// rollback, verilen kaynak listesini ters sırada geri alır.
func (e *Engine) rollback(resources []ApplyableResource) {
	// Ters sırada git
	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		if rev, ok := res.(Revertable); ok {
			fmt.Printf("Visualizing Rollback for %s...\n", res.GetName())
			if err := rev.Revert(e.Context); err != nil {
				fmt.Printf("❌ Failed to revert %s: %v\n", res.GetName(), err)
			} else {
				fmt.Printf("↺ Reverted %s\n", res.GetName())
			}
		}
	}
}
