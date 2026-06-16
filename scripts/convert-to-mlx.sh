#!/bin/bash
# ================================================
# Cofiswarm - Convert GGUF to MLX Script
# Converts Qwen2.5-14B and Llama-3.1-8B to MLX
# ================================================

set -e  # Exit on error

echo "🚀 Starting GGUF → MLX conversion..."

# Activate environment
conda activate mlx-env

# Define paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="/Users/Shared/llama/models"

GGUF_QWEN="${BASE_DIR}/large/Qwen2.5-14B-Instruct-Q4_K_M.gguf"
GGUF_LLAMA="${BASE_DIR}/medium/Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf"

MLX_QWEN="${BASE_DIR}/MLX/large/Qwen2.5-14B-Instruct-4bit"
MLX_LLAMA="${BASE_DIR}/MLX/medium/Meta-Llama-3.1-8B-Instruct-4bit"

# Create output directories
mkdir -p "${BASE_DIR}/MLX/large" "${BASE_DIR}/MLX/medium"

echo "📂 Output directories ready."

# Function to convert model
convert_model() {
    local gguf_path="$1"
    local mlx_path="$2"
    local model_name="$3"

    if [ ! -f "$gguf_path" ]; then
        echo "❌ GGUF file not found: $gguf_path"
        echo "Please make sure the model was downloaded first."
        return 1
    fi

    echo "🔄 Converting $model_name to MLX (4-bit)..."
    python gguf_to_mlx.py \
        --gguf-path "$gguf_path" \
        --mlx-path "$mlx_path" \
        --quantize 4bit

    echo "✅ Successfully converted $model_name"
}

# Perform conversions
convert_model "$GGUF_QWEN" "$MLX_QWEN" "Qwen2.5-14B"
convert_model "$GGUF_LLAMA" "$MLX_LLAMA" "Llama-3.1-8B"

echo ""
echo "🎉 All conversions completed!"
echo "📍 MLX models are ready at:"
echo "   • $MLX_QWEN"
echo "   • $MLX_LLAMA"
echo ""
echo "Next step: Run './open-swarmconfig-editor.sh' to update your agent model paths."
