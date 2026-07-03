package software

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const zipUTF8Flag = 1 << 11 // zip 文件头中标记文件名使用 UTF-8 的位

// ExtractZip 将指定的 zip 压缩包解压到 destDir 目录。
// 会自动处理中文文件名编码，并尝试去掉所有条目共有的顶层目录，直接解压到目标根目录。
// 解压完成后会删除原始的 zip 文件。
func ExtractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip failed: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir failed: %w", err)
	}

	prefix := commonTopLevelDir(r.File)

	for _, f := range r.File {
		name := decodeZipFilename(f)
		name = stripPrefix(name, prefix)
		if name == "" {
			continue
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(name))

		// 防止 zip Slip 路径穿越攻击
		cleanDest, err := filepath.Abs(destPath)
		if err != nil {
			return fmt.Errorf("invalid zip entry: %s", name)
		}
		cleanBase, err := filepath.Abs(destDir)
		if err != nil {
			return fmt.Errorf("invalid dest dir: %s", destDir)
		}
		if !strings.HasPrefix(cleanDest, cleanBase+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip entry: %s", name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	_ = os.Remove(zipPath)
	return nil
}

// decodeZipFilename 根据 zip 文件头的编码标志解码文件名。
// 若未设置 UTF-8 标志，则按 GBK 解码（兼容 Windows 中文 zip）。
func decodeZipFilename(f *zip.File) string {
	if f.Flags&zipUTF8Flag != 0 {
		return f.Name
	}
	decoder := simplifiedchinese.GBK.NewDecoder()
	decoded, _, err := transform.String(decoder, f.Name)
	if err != nil {
		return f.Name
	}
	return decoded
}

// commonTopLevelDir 返回所有文件条目共有的顶层目录前缀。
// 例如 zip 中所有文件都在 "resources/" 下，则返回 "resources/"，解压时会去掉这一层。
func commonTopLevelDir(files []*zip.File) string {
	var prefix string
	for i, f := range files {
		name := decodeZipFilename(f)
		name = strings.TrimPrefix(name, "/")
		if f.FileInfo().IsDir() && strings.HasSuffix(name, "/") {
			name = strings.TrimSuffix(name, "/")
		}
		idx := strings.Index(name, "/")
		var dir string
		if idx >= 0 {
			dir = name[:idx+1]
		} else {
			dir = name
			if f.FileInfo().IsDir() {
				dir += "/"
			} else {
				// 根目录下存在文件，说明没有统一顶层目录
				return ""
			}
		}
		if i == 0 {
			prefix = dir
		} else if dir != prefix {
			return ""
		}
	}
	return prefix
}

// stripPrefix 去掉文件名中的共有顶层目录前缀。
func stripPrefix(name, prefix string) string {
	if prefix == "" {
		return name
	}
	name = strings.TrimPrefix(name, "/")
	if strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return name
}
