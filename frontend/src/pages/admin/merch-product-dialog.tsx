import { useEffect, useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { ImagePlus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { deleteAdminMerchImage, uploadAdminMerchImage } from '@/entities/merch/api'
import { useCreateMerch, useUpdateMerch } from '@/entities/merch/queries'
import type { MerchProduct, MerchProductInput } from '@/entities/merch/types'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'

export function MerchProductDialog({
  contestId,
  product,
  open,
  onOpenChange,
}: {
  contestId: string
  product: MerchProduct | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const create = useCreateMerch(contestId)
  const update = useUpdateMerch(contestId, product?.id ?? '')
  const client = useQueryClient()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [price, setPrice] = useState('100')
  const [discount, setDiscount] = useState('')
  const [stock, setStock] = useState('0')
  const [image, setImage] = useState<File | null>(null)
  const [imageBusy, setImageBusy] = useState(false)

  useEffect(() => {
    if (!open) return
    setTitle(product?.title ?? '')
    setDescription(product?.description ?? '')
    setPrice(String(product?.price_points ?? 100))
    setDiscount(product?.discount_price_points ? String(product.discount_price_points) : '')
    setStock(String(product?.stock_quantity ?? 0))
    setImage(null)
  }, [open, product])

  async function submit(event: FormEvent) {
    event.preventDefault()
    const numericPrice = Number(price)
    const numericDiscount = discount ? Number(discount) : null
    const numericStock = Number(stock)
    if (
      !title.trim() ||
      !description.trim() ||
      !Number.isInteger(numericPrice) ||
      numericPrice <= 0 ||
      !Number.isInteger(numericStock) ||
      numericStock < 0 ||
      (numericDiscount !== null &&
        (!Number.isInteger(numericDiscount) ||
          numericDiscount <= 0 ||
          numericDiscount >= numericPrice))
    ) {
      toast.error('Проверьте название, цену, скидку и остаток')
      return
    }
    const input: MerchProductInput = {
      title: title.trim(),
      description: description.trim(),
      price_points: numericPrice,
      discount_price_points: numericDiscount,
      stock_quantity: numericStock,
    }
    try {
      const saved = product ? await update.mutateAsync(input) : await create.mutateAsync(input)
      if (image) await uploadAdminMerchImage(contestId, saved.id, image)
      await client.invalidateQueries({ queryKey: ['admin', 'merch-products', contestId] })
      toast.success(product ? 'Товар обновлён' : 'Товар создан')
      onOpenChange(false)
    } catch {
      toast.error('Не удалось сохранить товар. Остаток не может быть меньше резерва.')
    }
  }

  async function removeImage(imageId: string) {
    if (!product) return
    setImageBusy(true)
    try {
      await deleteAdminMerchImage(contestId, product.id, imageId)
      await client.invalidateQueries({ queryKey: ['admin', 'merch-products', contestId] })
      toast.success('Изображение удалено')
      onOpenChange(false)
    } catch {
      toast.error('Не удалось удалить изображение')
    } finally {
      setImageBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[90vh] max-w-2xl overflow-y-auto"
        title={product ? 'Редактировать товар' : 'Новый товар'}
        description="Цена заказа фиксируется в момент резервирования."
      >
        <form className="flex flex-col gap-4" onSubmit={submit}>
          <Field label="Название" required>
            {(props) => (
              <Input
                {...props}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                maxLength={300}
              />
            )}
          </Field>
          <Field label="Описание" required>
            {(props) => (
              <Textarea
                {...props}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={5}
                maxLength={20_000}
              />
            )}
          </Field>
          <div className="grid gap-4 sm:grid-cols-3">
            <Field label="Цена, баллы" required>
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  min={1}
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                />
              )}
            </Field>
            <Field label="Цена со скидкой">
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  min={1}
                  value={discount}
                  onChange={(e) => setDiscount(e.target.value)}
                  placeholder="Без скидки"
                />
              )}
            </Field>
            <Field label="Остаток" required>
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  min={0}
                  value={stock}
                  onChange={(e) => setStock(e.target.value)}
                />
              )}
            </Field>
          </div>
          <Field label="Добавить изображение" description="JPG, PNG, WEBP или GIF, до 20 МБ.">
            {(props) => (
              <Input
                {...props}
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                onChange={(e) => setImage(e.target.files?.[0] ?? null)}
              />
            )}
          </Field>
          {!!product?.images.length && (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
              {product.images.map((item) => (
                <div
                  key={item.id}
                  className="relative overflow-hidden rounded-[12px] border border-border"
                >
                  {item.url ? (
                    <img src={item.url} alt={product.title} className="h-28 w-full object-cover" />
                  ) : (
                    <div className="flex h-28 items-center justify-center bg-surface-2 text-muted">
                      <ImagePlus className="h-6 w-6" />
                    </div>
                  )}
                  <button
                    type="button"
                    disabled={imageBusy}
                    onClick={() => void removeImage(item.id)}
                    className="absolute right-2 top-2 rounded-lg bg-surface/90 p-1.5 text-danger shadow"
                    aria-label="Удалить изображение"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={create.isPending || update.isPending}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
