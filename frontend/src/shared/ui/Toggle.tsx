import { Field, Label, Switch } from '@headlessui/react'

export function Toggle({ label, description, checked, onChange }: { label: string; description?: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <Field className="flex items-center justify-between gap-4 rounded-lg border border-stone-300 bg-white px-3 py-3">
      <div><Label className="block text-xs font-semibold leading-relaxed text-stone-800">{label}</Label>{description ? <p className="mb-0 mt-1 text-[10px] leading-relaxed text-muted">{description}</p> : null}</div>
      <Switch checked={checked} onChange={onChange} className="group inline-flex h-6 w-11 shrink-0 items-center rounded-full bg-stone-300 transition data-checked:bg-emerald-700">
        <span className="size-4 translate-x-1 rounded-full bg-white transition group-data-checked:translate-x-6" />
      </Switch>
    </Field>
  )
}
