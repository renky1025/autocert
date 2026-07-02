#!/bin/bash

# AutoCert 一键打包脚本 - Linux 版本
# 格式: autocert_${VERSION}_linux_${ARCH}.tar.gz
#       autocert_${VERSION}_windows_${ARCH}.zip

set -euo pipefail

# 脚本参数
VERSION=${1:-"dev"}
DIST_DIR=${2:-"dist"}
BINARY_NAME=${3:-"autocert"}
PLATFORM=${4:-"all"}  # all, linux, windows, darwin

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1" >&2
}

# 检查必需工具
check_tools() {
    log_info "检查必需工具..."
    
    local missing_tools=()
    
    if ! command -v go >/dev/null; then
        missing_tools+=("go")
    fi
    
    if ! command -v tar >/dev/null; then
        missing_tools+=("tar")
    fi
    
    if ! command -v zip >/dev/null; then
        missing_tools+=("zip")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "缺少必需工具: ${missing_tools[*]}"
        log_error "请安装缺少的工具后重试"
        exit 1
    fi
    
    log_info "工具检查通过"
}

# 获取构建标志
get_build_flags() {
    local build_time=$(date -u '+%Y-%m-%d_%H:%M:%S')
    local commit_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    echo "-X main.version=${VERSION} -X main.buildTime=${build_time} -X main.commitHash=${commit_hash}"
}

# 构建二进制文件
build_binary() {
    local goos=$1
    local goarch=$2
    local output_path=$3
    local ldflags=$4
    
    log_info "构建 ${goos}/${goarch}..."
    
    GOOS=${goos} GOARCH=${goarch} go build -ldflags "${ldflags}" -o "${output_path}" .
    
    if [ $? -eq 0 ]; then
        log_info "构建成功: ${output_path}"
    else
        log_error "构建失败: ${goos}/${goarch}"
        return 1
    fi
}

# 创建 Linux 发布包
package_linux() {
    local arch=$1
    local build_flags=$2
    
    log_info "打包 Linux ${arch}..."
    
    local binary_path="${DIST_DIR}/${BINARY_NAME}"
    local package_name="${BINARY_NAME}_${VERSION}_linux_${arch}.tar.gz"
    local package_path="${DIST_DIR}/${package_name}"
    
    # 构建二进制文件
    build_binary "linux" "${arch}" "${binary_path}" "${build_flags}"
    
    # 创建 tar.gz 包
    tar -czf "${package_path}" -C "${DIST_DIR}" "${BINARY_NAME}"
    
    # 清理临时文件
    rm -f "${binary_path}"
    
    log_info "Linux ${arch} 打包完成: ${package_name}"
    echo "  📦 ${package_path}"
}

# 创建 Windows 发布包
package_windows() {
    local arch=$1
    local build_flags=$2
    
    log_info "打包 Windows ${arch}..."
    
    local binary_path="${DIST_DIR}/${BINARY_NAME}.exe"
    local package_name="${BINARY_NAME}_${VERSION}_windows_${arch}.zip"
    local package_path="${DIST_DIR}/${package_name}"
    
    # 构建二进制文件
    build_binary "windows" "${arch}" "${binary_path}" "${build_flags}"
    
    # 创建 zip 包
    cd "${DIST_DIR}"
    zip "${package_name}" "${BINARY_NAME}.exe"
    cd - > /dev/null
    
    # 清理临时文件
    rm -f "${binary_path}"
    
    log_info "Windows ${arch} 打包完成: ${package_name}"
    echo "  📦 ${package_path}"
}

# 创建 macOS 发布包
package_darwin() {
    local arch=$1
    local build_flags=$2
    
    log_info "打包 macOS ${arch}..."
    
    local binary_path="${DIST_DIR}/${BINARY_NAME}"
    local package_name="${BINARY_NAME}_${VERSION}_darwin_${arch}.tar.gz"
    local package_path="${DIST_DIR}/${package_name}"
    
    # 构建二进制文件
    build_binary "darwin" "${arch}" "${binary_path}" "${build_flags}"
    
    # 创建 tar.gz 包
    tar -czf "${package_path}" -C "${DIST_DIR}" "${BINARY_NAME}"
    
    # 清理临时文件
    rm -f "${binary_path}"
    
    log_info "macOS ${arch} 打包完成: ${package_name}"
    echo "  📦 ${package_path}"
}

# 主函数
main() {
    log_info "AutoCert 一键打包工具"
    log_info "版本: ${VERSION}"
    log_info "输出目录: ${DIST_DIR}"
    log_info "二进制名称: ${BINARY_NAME}"
    log_info "平台: ${PLATFORM}"
    
    # 检查工具
    check_tools
    
    # 创建输出目录
    mkdir -p "${DIST_DIR}"
    
    # 获取构建标志
    local build_flags=$(get_build_flags)
    log_debug "构建标志: ${build_flags}"
    
    # 根据平台参数进行打包
    case "${PLATFORM}" in
        "all")
            log_info "打包所有支持的平台..."
            
            # Linux 平台
            package_linux "amd64" "${build_flags}"
            package_linux "arm64" "${build_flags}"
            
            # Windows 平台
            package_windows "amd64" "${build_flags}"
            package_windows "arm64" "${build_flags}"
            
            # macOS 平台
            package_darwin "amd64" "${build_flags}"
            package_darwin "arm64" "${build_flags}"
            ;;
        "linux")
            log_info "打包 Linux 平台..."
            package_linux "amd64" "${build_flags}"
            package_linux "arm64" "${build_flags}"
            ;;
        "windows")
            log_info "打包 Windows 平台..."
            package_windows "amd64" "${build_flags}"
            package_windows "arm64" "${build_flags}"
            ;;
        "darwin"|"macos")
            log_info "打包 macOS 平台..."
            package_darwin "amd64" "${build_flags}"
            package_darwin "arm64" "${build_flags}"
            ;;
        *)
            log_error "不支持的平台: ${PLATFORM}"
            log_error "支持的平台: all, linux, windows, darwin"
            exit 1
            ;;
    esac
    
    # 显示结果
    echo ""
    log_info "打包完成! 生成的文件:"
    ls -la "${DIST_DIR}"/*.tar.gz "${DIST_DIR}"/*.zip 2>/dev/null || true
    
    # 统计信息
    local total_files=$(ls "${DIST_DIR}"/*.tar.gz "${DIST_DIR}"/*.zip 2>/dev/null | wc -l)
    local total_size=$(du -sh "${DIST_DIR}" | cut -f1)
    
    echo ""
    echo "📊 统计信息:"
    echo "  📁 输出目录: ${DIST_DIR}"
    echo "  📦 发布包数量: ${total_files}"
    echo "  💾 总大小: ${total_size}"
    echo ""
    echo "🎉 所有发布包已准备就绪!"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
