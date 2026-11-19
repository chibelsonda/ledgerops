import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Input, PasswordInput, Select, Textarea } from './FormControls'

describe('validated form controls', () => {
  it('renders every invalid control with a red border', () => {
    render(
      <>
        <Input label="Email" error="Enter a valid email." />
        <Select label="Status" error="Select a status."><option value="">Choose</option></Select>
        <Textarea label="Audit note" error="Enter an audit note." />
      </>,
    )

    for (const control of [
      screen.getByRole('textbox', { name: 'Email' }),
      screen.getByRole('combobox', { name: 'Status' }),
      screen.getByRole('textbox', { name: 'Audit note' }),
    ]) {
      expect(control).toHaveStyle({ borderColor: '#dc2626' })
      expect(control).toHaveAttribute('aria-invalid', 'true')
    }
  })

  it('toggles password visibility with an accessible button', () => {
    render(<PasswordInput label="Password" defaultValue="secure password" />)

    const input = screen.getByLabelText('Password')
    const toggle = screen.getByRole('button', { name: 'Show password' })
    expect(input).toHaveAttribute('type', 'password')

    fireEvent.click(toggle)
    expect(input).toHaveAttribute('type', 'text')
    expect(screen.getByRole('button', { name: 'Hide password' })).toHaveAttribute('aria-pressed', 'true')
  })
})
