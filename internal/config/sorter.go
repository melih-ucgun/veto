package config

import (
	"fmt"
	"strings"
)

// SortResources, kaynakları bağımlılıklarına göre sıralar ve katmanlara ayırır.
// Kahn's Algorithm kullanır.
func SortResources(resources []ResourceConfig) ([][]ResourceConfig, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	// 1. Kaynak haritası ve bağımlılık grafiği oluştur
	resourceMap := make(map[string]ResourceConfig)
	graph := make(map[string][]string) // key -> dependants (key'e bağımlı olanlar)
	inDegree := make(map[string]int)   // key -> dependency count (key kaç şeye bağımlı)

	// ID çakışması kontrolü ve hazırlık
	for _, res := range resources {
		if _, exists := resourceMap[res.ID]; exists {
			return nil, fmt.Errorf("duplicate resource ID found: %s", res.ID)
		}
		resourceMap[res.ID] = res
		inDegree[res.ID] = 0 // Başlangıç değeri
	}

	// 2. Grafiği doldur
	for _, res := range resources {
		for _, depID := range res.DependsOn {
			// Bağımlı olunan kaynak var mı?
			if _, exists := resourceMap[depID]; !exists {
				return nil, fmt.Errorf("resource '%s' depends on unknown resource '%s'", res.ID, depID)
			}

			// Graph: depID -> res.ID (depID tamamlanınca res.ID açılabilir)
			graph[depID] = append(graph[depID], res.ID)
			inDegree[res.ID]++
		}
	}

	// 3. Başlangıç seti (Bağımsız düğümler - Layer 0)
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Lexicographical sortQueue gerekebilir mi? Şimdilik gerek yok ama
	// deterministik olması için queue sıralanabilir.

	var layers [][]ResourceConfig
	processedCount := 0

	for len(queue) > 0 {
		var nextLayer []string
		var currentLayerConfigs []ResourceConfig

		// Mevcut queue, bir katmanı temsil eder (parallelizable)
		// Ancak Kahn algoritmasının standart hali node node ilerler.
		// Paralel katmanlar oluşturmak için, şu anki queue'yu "dondurup" işlemeliyiz.

		layerSize := len(queue)
		for i := 0; i < layerSize; i++ {
			node := queue[i]
			processedCount++
			currentLayerConfigs = append(currentLayerConfigs, resourceMap[node])

			// Node'a bağımlı olanların derecesini düşür
			for _, neighbour := range graph[node] {
				inDegree[neighbour]--
				if inDegree[neighbour] == 0 {
					nextLayer = append(nextLayer, neighbour)
				}
			}
		}

		layers = append(layers, currentLayerConfigs)
		queue = nextLayer // Bir sonraki katmana geç
	}

	// 4. Döngü kontrolü
	if processedCount != len(resources) {
		// Döngü var demektir. Hangi node'lar işlenmedi?
		var unprocessed []string
		for id, degree := range inDegree {
			if degree > 0 {
				unprocessed = append(unprocessed, id)
			}
		}
		return nil, fmt.Errorf("dependency cycle detected involves: %v", strings.Join(unprocessed, ", "))
	}

	return layers, nil
}
