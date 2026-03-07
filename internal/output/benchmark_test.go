package output

import (
	"fmt"
	"io"
	"testing"
)

type benchmarkSale struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	ProductName string `json:"product_name"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
	PriceCents  int    `json:"price_cents"`
	FeeCents    int    `json:"fee_cents"`
}

func BenchmarkStreamJSONWithJQ(b *testing.B) {
	for _, size := range []int{50, 500} {
		sales := benchmarkSales(size)
		writeItems := writeBenchmarkSales(sales)

		b.Run(fmt.Sprintf("%d_items", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				err := streamJSONWithJQ(io.Discard, "sales", `.sales[] | {
  id,
  email,
  status,
  net_cents: (.price_cents - .fee_cents)
}`, writeItems)
				if err != nil {
					b.Fatalf("streamJSONWithJQ failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTableRender(b *testing.B) {
	for _, size := range []int{50, 500} {
		table := benchmarkSalesTable(size)

		b.Run(fmt.Sprintf("%d_rows", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := table.Render(io.Discard); err != nil {
					b.Fatalf("Render failed: %v", err)
				}
			}
		})
	}
}

func benchmarkSales(count int) []benchmarkSale {
	sales := make([]benchmarkSale, 0, count)
	for i := 0; i < count; i++ {
		sales = append(sales, benchmarkSale{
			ID:          fmt.Sprintf("sale_%04d", i),
			Email:       fmt.Sprintf("buyer-%04d@example.com", i),
			ProductName: fmt.Sprintf("Illustration Bundle %04d", i),
			CreatedAt:   "2026-03-08",
			Status:      benchmarkSaleStatus(i),
			PriceCents:  1500 + (i % 7 * 100),
			FeeCents:    150 + (i % 3 * 10),
		})
	}
	return sales
}

func benchmarkSaleStatus(index int) string {
	switch index % 3 {
	case 0:
		return "paid"
	case 1:
		return "refunded"
	default:
		return "pending"
	}
}

func writeBenchmarkSales(sales []benchmarkSale) func(func(any) error) error {
	return func(writeItem func(any) error) error {
		for _, sale := range sales {
			if err := writeItem(sale); err != nil {
				return err
			}
		}
		return nil
	}
}

func benchmarkSalesTable(count int) *Table {
	style := Styler{enabled: true}
	table := NewTable("ID", "PRODUCT", "STATUS", "EMAIL", "DATE", "NET")
	table.SetStyler(style)

	for _, sale := range benchmarkSales(count) {
		table.AddRow(
			sale.ID,
			sale.ProductName+" with extended commercial license",
			benchmarkStatusCell(style, sale.Status),
			sale.Email,
			sale.CreatedAt,
			fmt.Sprintf("$%.2f", float64(sale.PriceCents-sale.FeeCents)/100),
		)
	}

	return table
}

func benchmarkStatusCell(style Styler, status string) string {
	switch status {
	case "paid":
		return style.Green(status)
	case "refunded":
		return style.Yellow(status)
	default:
		return style.Dim(status)
	}
}
