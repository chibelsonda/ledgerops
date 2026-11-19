import { Field, Input as HeadlessInput, Label, Select as HeadlessSelect, Textarea as HeadlessTextarea } from '@headlessui/react'
import { ChevronDown, Eye, EyeOff } from 'lucide-react'
import { useState } from 'react'
import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

const controlClass = 'form-control w-full rounded-lg border border-stone-300 bg-white px-3 py-2.5 text-xs leading-5 text-stone-900 outline-none transition placeholder:text-stone-500 focus:border-emerald-600 focus:ring-3 focus:ring-emerald-700/10'
const invalidBorder = '#dc2626'

type InputProps = InputHTMLAttributes<HTMLInputElement> & { label: string; icon?: ReactNode; endAdornment?: ReactNode; hideLabel?: boolean; error?: string }

export function Input({ label, icon, endAdornment, hideLabel, error, className = '', ...props }: InputProps) {
  return (
    <Field className="relative w-full">
      <Label className={hideLabel ? 'sr-only' : 'mb-2 block text-xs font-semibold leading-relaxed text-stone-800'}>{label}</Label>
      <div className="relative">
        {icon ? <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-stone-500">{icon}</span> : null}
        <HeadlessInput {...props} invalid={Boolean(error)} style={error ? { ...props.style, borderColor: invalidBorder } : props.style} className={`${controlClass} ${icon ? 'pl-9' : ''} ${endAdornment ? 'pr-11' : ''} ${className}`} />
        {endAdornment ? <span className="absolute right-2 top-1/2 -translate-y-1/2">{endAdornment}</span> : null}
      </div>
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}

type PasswordInputProps = Omit<InputProps, 'type' | 'endAdornment'>

export function PasswordInput({ label, ...props }: PasswordInputProps) {
  const [visible, setVisible] = useState(false)
  const action = visible ? 'Hide' : 'Show'
  return (
    <Input
      {...props}
      label={label}
      type={visible ? 'text' : 'password'}
      endAdornment={(
        <button
          type="button"
          className="grid size-8 place-items-center rounded-lg text-stone-500 transition hover:bg-stone-100 hover:text-stone-800 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-emerald-700"
          aria-label={`${action} ${label.toLowerCase()}`}
          aria-pressed={visible}
          onClick={() => setVisible((value) => !value)}
        >
          {visible ? <EyeOff size={17} aria-hidden="true" /> : <Eye size={17} aria-hidden="true" />}
        </button>
      )}
    />
  )
}

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & { label: string; children: ReactNode; hideLabel?: boolean; error?: string }

export function Select({ label, children, hideLabel, error, className = '', ...props }: SelectProps) {
  return (
    <Field className="relative">
      <Label className={hideLabel ? 'sr-only' : 'mb-2 block text-xs font-semibold leading-relaxed text-stone-800'}>{label}</Label>
      <div className="relative">
        <HeadlessSelect {...props} invalid={Boolean(error)} style={error ? { ...props.style, borderColor: invalidBorder } : props.style} className={`${controlClass} appearance-none pr-9 ${className}`}>{children}</HeadlessSelect>
        <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-stone-500" size={14} />
      </div>
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string; error?: string }

export function Textarea({ label, error, className = '', ...props }: TextareaProps) {
  return (
    <Field>
      <Label className="mb-2 block text-xs font-semibold leading-relaxed text-stone-800">{label}</Label>
      <HeadlessTextarea {...props} invalid={Boolean(error)} style={error ? { ...props.style, borderColor: invalidBorder } : props.style} className={`${controlClass} min-h-28 resize-y leading-relaxed ${className}`} />
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}
