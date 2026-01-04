package service

import (
	"fmt"
	"idcard/internal/model"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/tigorlazuardi/tanggal"
)

type (
	PdfService interface {
		PrintPDF(user *model.User, outputPath string) error
	}

	pdfSvc struct{}
)

func NewPdfService() PdfService {
	return &pdfSvc{}
}

func (s *pdfSvc) PrintPDF(user *model.User, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Formulir Pendaftaran Penyetor Afval", false)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "Formulir Pendaftaran Penyetor Afval", "", 1, "C", false, 0, "")
	pdf.Ln(0)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 10, "PT. Sinar Indah Kertas", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, "Identitas Penyetor Afval")
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("Nama     : %s", user.Name))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("NIK         : %s", user.NIK))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("No. Telp  : %s", user.Phone))
	pdf.Ln(8)
	pdf.MultiCell(0, 8, fmt.Sprintf("Alamat    : %s", user.Address), "", "", false)
	pdf.Ln(8)

	// Declaration statements
	statements := []string{
		"Dengan menandatangani formulir ini saya menyatakan:",
		"    1. Saya mengajukan/mendaftar sebagai penyetor afval PT. Sinar Indah Kertas.",
		"    2. Afval yang saya setor adalah hasil kegiatan yang sah dan tidak melanggar hukum.",
		"    3. Saya berkomitmen untuk menjaga kualitas dan kejujuran dalam setiap setoran.",
		"    4. Saya bersedia menjalani proses inspeksi & verifikasi sesuai sistem QC yang diterapkan.",
		"  5. PT. Sinar Indah Kertas berhak menolak apabila kualitas afval tidak memenuhi standar            perusahaan.",
		"    6. Saya bersedia mengikuti tata tertib yang berlaku, di antaranya:",
		"       a. Tidak mengambil gambar/foto/video di area perusahaan.",
		"       b. Tidak merokok di area perusahaan.",
		"       c. Tidak melanggar batas kecepatan kendaraan di area perusahaan.",
		"   7. Saya menyadari bahwa pelanggaran terhadap komitmen dapat berdampak pada pemutusan            kerjasama.",
		"  8. Saya menyatakan bahwa data & pernyataan yang saya berikan adalah benar dan dapat             dipertanggungjawabkan.",
	}

	for _, line := range statements {
		pdf.MultiCell(0, 8, line, "", "", false)
	}
	pdf.Ln(10)

	tgl, _ := tanggal.Papar(time.Now(), "Kudus", tanggal.WIB)
	format := []tanggal.Format{
		tanggal.LokasiDenganKoma,
		tanggal.Hari,
		tanggal.NamaBulan,
		tanggal.Tahun,
	}

	// Signature block
	pdf.CellFormat(0, 6, tgl.Format(" ", format), "", 1, "R", false, 0, "")
	pdf.Ln(18)
	pdf.CellFormat(0, 6, user.Name, "", 1, "R", false, 0, "")

	// Output
	return pdf.OutputFileAndClose(outputPath)
}
