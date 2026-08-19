/** Center-crop an image into a square PNG of the given pixel size. */
export async function resizeImageToSquare(file: File, size = 96): Promise<File> {
  const url = URL.createObjectURL(file)
  try {
    const image = await loadImage(url)
    const canvas = document.createElement('canvas')
    canvas.width = size
    canvas.height = size
    const context = canvas.getContext('2d')
    if (!context) throw new Error('canvas')
    const scale = Math.max(size / image.width, size / image.height)
    const width = image.width * scale
    const height = image.height * scale
    context.drawImage(image, (size - width) / 2, (size - height) / 2, width, height)
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
    if (!blob) throw new Error('blob')
    return new File([blob], 'icon.png', { type: 'image/png' })
  } finally {
    URL.revokeObjectURL(url)
  }
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('image'))
    image.src = src
  })
}
