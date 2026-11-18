import { expect, test } from '@playwright/test'

async function signIn(page: import('@playwright/test').Page) {
  await page.goto('/')
  await page.getByLabel('Email address').fill('mara@ledgerops.demo')
  await page.getByLabel('Password').fill('LedgerOpsDemo!2026')
  const loginResponse = page.waitForResponse((response) => response.url().endsWith('/api/v1/auth/login') && response.request().method() === 'POST')
  await page.getByRole('button', { name: 'Sign in securely' }).click()
  const session = await (await loginResponse).json() as { csrfToken: string }
  await expect(page.getByRole('heading', { name: 'The numbers that need your attention.' })).toBeVisible()
  return session.csrfToken
}

async function signInAs(page: import('@playwright/test').Page, email: string, password: string) {
  await page.goto('/')
  await page.getByLabel('Email address').fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in securely' }).click()
  await expect(page.getByRole('heading', { name: 'The numbers that need your attention.' })).toBeVisible()
}

async function signOut(page: import('@playwright/test').Page) {
  await page.locator('.user-menu-trigger').click()
  await page.locator('.user-menu-signout').click()
  await expect(page.getByRole('heading', { name: 'Sign in to LedgerOps' })).toBeVisible()
}

test('signs in and ends the server-side session', async ({ page }) => {
  await signIn(page)
  await signOut(page)
})

test('hides operator tools from reviewers', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Email address').fill('reviewer@ledgerops.demo')
  await page.getByLabel('Password').fill('LedgerOpsReview!2026')
  await page.getByRole('button', { name: 'Sign in securely' }).click()
  await expect(page.getByRole('heading', { name: 'The numbers that need your attention.' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Data sync' })).toHaveCount(0)
  await page.getByRole('link', { name: 'Invoices' }).click()
  await expect(page.getByRole('button', { name: 'New invoice' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /^Edit / })).toHaveCount(0)
  await page.getByRole('link', { name: 'Projects' }).click()
  await expect(page.getByRole('button', { name: 'New project' })).toHaveCount(0)
  await page.getByRole('link', { name: 'Month-end close' }).click()
  await expect(page.getByRole('button', { name: 'New close period' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Complete', exact: true })).toHaveCount(0)
  await page.getByRole('link', { name: 'Cash forecast' }).click()
  await expect(page.getByRole('button', { name: 'New scenario' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Add adjustment' })).toHaveCount(0)
  await page.goto('/sync')
  await expect(page.getByRole('heading', { name: 'The numbers that need your attention.' })).toBeVisible()
})

test('shows the financial overview and invoice data', async ({ page }) => {
  await signIn(page)
  await expect(page.locator('.app-shell')).toHaveCSS('display', 'grid')
  await expect(page.locator('.sidebar')).toHaveCSS('background-color', 'rgb(20, 39, 31)')
  await expect(page.locator('body')).toHaveCSS('font-family', /Manrope Variable/)
  await expect(page.locator('.page-header h1')).toHaveCSS('font-family', /Manrope Variable/)
  await expect(page.locator('.metric-card').first()).toHaveCSS('border-radius', '8px')
  await expect(page.getByRole('heading', { name: 'The numbers that need your attention.' })).toBeVisible()
  await expect(page.getByText('Outstanding A/R')).toBeVisible()

  const fontsLoaded = await page.evaluate(() => ({
    display: document.fonts.check('36px "Newsreader Variable"'),
    sans: document.fonts.check('16px "Manrope Variable"'),
  }))
  expect(fontsLoaded).toEqual({ display: true, sans: true })

  await page.getByRole('link', { name: 'Invoices' }).click()
  await expect(page.getByRole('heading', { name: 'Invoices' })).toBeVisible()
  await page.getByPlaceholder('Search client or invoice…').fill('INV-2026-055')
  await expect(page.getByText('INV-2026-055')).toBeVisible()
})

test('creates and edits an invoice with an audit trail', async ({ page }) => {
  const invoiceNumber = `INV-E2E-${Date.now()}`
  const csrfToken = await signIn(page)
  await page.getByRole('link', { name: 'Invoices' }).click()

  await page.getByRole('button', { name: 'New invoice' }).click()
  await page.getByLabel('Invoice number').fill(invoiceNumber)
  await page.getByLabel('Client', { exact: true }).selectOption({ label: 'Arc & Co.' })
  await page.getByLabel('Status', { exact: true }).selectOption('open')
  await page.getByLabel('Amount', { exact: true }).fill('1234.56')
  await page.getByRole('button', { name: 'Create invoice' }).click()
  await expect(page.getByText(invoiceNumber, { exact: true })).toBeVisible()

  await page.getByRole('button', { name: `Edit ${invoiceNumber}` }).click()
  await expect(page.getByRole('heading', { name: `Edit ${invoiceNumber}` })).toBeVisible()
  await page.getByRole('tab', { name: 'Audit history' }).click()
  await expect(page.getByText('Created', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'Details' }).click()
  await page.getByLabel('Amount', { exact: true }).fill('1300')
  await page.getByLabel('Audit note', { exact: true }).fill('Updated after confirming the final client scope.')
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByText('$1,300.00', { exact: true }).first()).toBeVisible()

  await page.getByRole('button', { name: `Edit ${invoiceNumber}` }).click()
  await page.getByRole('tab', { name: 'Audit history' }).click()
  await expect(page.getByText('Updated after confirming the final client scope.')).toBeVisible()

  const overpaymentStatus = await page.evaluate(async ({ number, csrfToken }) => {
    const listResponse = await fetch(`/api/v1/invoices?query=${encodeURIComponent(number)}&page=1&pageSize=5`)
    const list = await listResponse.json() as { items: Array<{ id: string }> }
    const response = await fetch(`/api/v1/invoices/${list.items[0].id}/payments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({
        amountMinor: 130001,
        receivedOn: new Date().toISOString().slice(0, 10),
        reference: 'OVERPAYMENT-CHECK',
        auditNote: 'This request must be rejected.',
      }),
    })
    return response.status
  }, { number: invoiceNumber, csrfToken })
  expect(overpaymentStatus).toBe(422)

  await page.getByRole('tab', { name: 'Payments' }).click()
  await page.getByLabel('Payment amount').fill('300')
  await page.getByLabel('Payment reference').fill('BANK-PARTIAL')
  await page.getByLabel('Payment audit note').fill('Matched the partial deposit to the bank statement.')
  await page.getByRole('button', { name: 'Record payment' }).click()
  await expect(page.getByText('$1,000.00 outstanding')).toBeVisible()
  await expect(page.getByText('BANK-PARTIAL')).toBeVisible()
  await page.getByRole('tab', { name: 'Details' }).click()
  await expect(page.getByRole('dialog').getByText('Partial', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'Audit history' }).click()
  await expect(page.getByText('Payment Recorded', { exact: true })).toBeVisible()

  await page.getByRole('tab', { name: 'Payments' }).click()
  await page.getByLabel('Payment amount').fill('1000')
  await page.getByLabel('Payment reference').fill('BANK-FINAL')
  await page.getByLabel('Payment audit note').fill('Matched the final deposit and closed the balance.')
  await page.getByRole('button', { name: 'Record payment' }).click()
  await expect(page.getByText('$0.00 outstanding')).toBeVisible()
  await expect(page.getByText('This invoice is fully paid.')).toBeVisible()

  await page.getByRole('tab', { name: 'Details' }).click()
  await page.getByLabel('Amount', { exact: true }).fill('1200')
  await page.getByLabel('Audit note', { exact: true }).fill('Attempting to reduce the invoice below collected cash.')
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByRole('alert').filter({ hasText: 'conflicts with the invoice balance or status' })).toBeVisible()
  await page.getByRole('button', { name: 'Close invoice form' }).click()

  const invoiceRow = page.getByRole('row').filter({ hasText: invoiceNumber })
  await expect(invoiceRow.getByText('Paid', { exact: true })).toBeVisible()
  await expect(invoiceRow.getByText('$0.00', { exact: true })).toBeVisible()
})

test('paginates and filters invoices on the server', async ({ page }) => {
  await signIn(page)
  await page.getByRole('link', { name: 'Invoices' }).click()
  await page.getByLabel('Rows per page').selectOption('5')
  await expect(page.getByText(/Page 1 of (?:[2-9]|[1-9]\d+)/)).toBeVisible()
  await page.getByRole('button', { name: 'Next invoice page' }).click()
  await expect(page.getByText(/Page 2 of (?:[2-9]|[1-9]\d+)/)).toBeVisible()

  await page.getByPlaceholder('Search client or invoice…').fill('INV-2026-055')
  await expect(page.getByText('INV-2026-055')).toBeVisible()
  await expect(page.getByText('Page 1 of 1')).toBeVisible()
  await page.getByPlaceholder('Search client or invoice…').fill('')
  await page.getByLabel('Filter by status').selectOption('paid')
  await expect(page.locator('tbody').getByText('Paid', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('INV-2026-055')).toHaveCount(0)
})

test('approves immutable credit notes and payment reversals with separation of duties', async ({ page }) => {
  const invoiceNumber = `INV-ADJ-${Date.now()}`
  const controllerCSRF = await signIn(page)
  await page.getByRole('link', { name: 'Invoices' }).click()
  await page.getByRole('button', { name: 'New invoice' }).click()
  await page.getByLabel('Invoice number').fill(invoiceNumber)
  await page.getByLabel('Client', { exact: true }).selectOption({ label: 'Arc & Co.' })
  await page.getByLabel('Status', { exact: true }).selectOption('open')
  await page.getByLabel('Amount', { exact: true }).fill('1000')
  await page.getByRole('button', { name: 'Create invoice' }).click()

  await page.getByPlaceholder('Search client or invoice…').fill(invoiceNumber)
  await page.getByRole('button', { name: `Edit ${invoiceNumber}` }).click()
  await page.getByRole('tab', { name: 'Payments' }).click()
  await page.getByLabel('Payment amount').fill('400')
  await page.getByLabel('Payment reference').fill('ADJ-PAYMENT')
  await page.getByLabel('Payment audit note').fill('Matched payment before testing controlled adjustments.')
  await page.getByRole('button', { name: 'Record payment' }).click()
  await expect(page.getByText('$600.00 outstanding')).toBeVisible()

  await page.getByRole('tab', { name: 'Credits & reversals' }).click()
  await page.getByLabel('Credit amount').fill('100')
  await page.getByLabel('Required justification').fill('Approved client service concession.')
  await page.getByRole('button', { name: 'Submit for approval' }).click()
  const creditRow = page.locator('.adjustment-row').filter({ hasText: 'Credit Note' })
  await expect(creditRow.getByText('Pending', { exact: true })).toBeVisible()
  await expect(creditRow.getByRole('button', { name: 'Approve' })).toHaveCount(0)

  await page.getByLabel('Adjustment type').selectOption('payment_reversal')
  await page.getByLabel('Payment to reverse').selectOption({ index: 1 })
  await page.getByLabel('Required justification').fill('Payment was duplicated in the bank import.')
  await page.getByRole('button', { name: 'Submit for approval' }).click()
  await expect(page.locator('.adjustment-row').filter({ hasText: 'Payment Reversal' }).getByText('Pending', { exact: true })).toBeVisible()

  const controllerDecisionStatus = await page.evaluate(async ({ invoiceNumber, controllerCSRF }) => {
    const invoices = await (await fetch(`/api/v1/invoices?query=${encodeURIComponent(invoiceNumber)}&page=1&pageSize=5`)).json() as { items: Array<{ id: string }> }
    const adjustments = await (await fetch(`/api/v1/invoices/${invoices.items[0].id}/adjustments`)).json() as { items: Array<{ id: string }> }
    return (await fetch(`/api/v1/adjustments/${adjustments.items[0].id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': controllerCSRF },
      body: JSON.stringify({ decision: 'approved', note: 'Controller must not approve this request.' }),
    })).status
  }, { invoiceNumber, controllerCSRF })
  expect(controllerDecisionStatus).toBe(403)

  await page.getByRole('button', { name: 'Close invoice form' }).click()
  await signOut(page)
  await signInAs(page, 'admin@ledgerops.demo', 'LedgerOpsAdmin!2026')
  await page.getByRole('link', { name: 'Invoices' }).click()
  await page.getByPlaceholder('Search client or invoice…').fill(invoiceNumber)
  await page.getByRole('button', { name: `Edit ${invoiceNumber}` }).click()
  await page.getByRole('tab', { name: 'Credits & reversals' }).click()

  const pendingCredit = page.locator('.adjustment-row').filter({ hasText: 'Credit Note' })
  await pendingCredit.getByRole('button', { name: 'Approve' }).click()
  await page.getByLabel('Approved review note').fill('Verified concession approval against the client record.')
  await page.getByRole('button', { name: 'Confirm approved' }).click()
  await expect(pendingCredit.getByText('Approved', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'Payments' }).click()
  await expect(page.getByText('$500.00 outstanding')).toBeVisible()

  await page.getByRole('tab', { name: 'Credits & reversals' }).click()
  const pendingReversal = page.locator('.adjustment-row').filter({ hasText: 'Payment Reversal' })
  await pendingReversal.getByRole('button', { name: 'Approve' }).click()
  await page.getByLabel('Approved review note').fill('Confirmed duplicate receipt against the bank statement.')
  await page.getByRole('button', { name: 'Confirm approved' }).click()
  await expect(pendingReversal.getByText('Approved', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'Payments' }).click()
  await expect(page.getByText('$900.00 outstanding')).toBeVisible()
  await expect(page.getByText(/Reversed/)).toBeVisible()
})

test('shows the safe QuickBooks setup state', async ({ page }) => {
  await signIn(page)
  await page.goto('/sync')
  await expect(page.getByRole('heading', { name: 'Data synchronization' })).toBeVisible()
  await expect(page.getByText('Setup required')).toBeVisible()
  await expect(page.getByRole('button', { name: /Connect QuickBooks/ })).toBeDisabled()
  await expect(page.getByRole('button', { name: /Run sandbox sync/ })).toBeDisabled()
})

test('manages client contacts and queues an overdue reminder idempotently', async ({ page }) => {
  const suffix = Date.now()
  const clientName = `Reminder Client ${suffix}`
  const invoiceNumber = `INV-REM-${suffix}`
  const csrfToken = await signIn(page)
  await page.getByRole('link', { name: 'Clients' }).click()
  await page.getByRole('button', { name: 'Add client' }).click()
  await page.getByLabel('Client name').fill(clientName)
  await page.getByLabel('Payment terms (days)').fill('15')
  await page.getByLabel('Days after due date').fill('0')
  await page.getByLabel('Contact name').fill('Avery Stone')
  await page.getByLabel('Email address').fill(`billing-${suffix}@example.com`)
  await page.getByLabel('Role').fill('Finance lead')
  await page.getByRole('button', { name: 'Create client' }).click()
  await page.getByPlaceholder('Search client, contact, or email…').fill(clientName)
  await expect(page.getByText(clientName, { exact: true })).toBeVisible()

  await page.getByRole('row').filter({ hasText: clientName }).getByRole('button', { name: 'Manage' }).click()
  await page.getByLabel('Contact name').fill('Jordan Lee')
  await page.getByLabel('Contact email').fill(`ap-${suffix}@example.com`)
  await page.getByLabel('Role').fill('Accounts payable')
  await page.getByRole('button', { name: 'Add contact' }).click()
  await expect(page.getByText('Jordan Lee', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Close client form' }).click()

  const invoiceStatus = await page.evaluate(async ({ clientName, invoiceNumber, csrfToken }) => {
    const clients = await (await fetch(`/api/v1/clients?query=${encodeURIComponent(clientName)}&page=1&pageSize=5`)).json() as { items: Array<{ id: string }> }
    const today = new Date(); const issued = new Date(today); issued.setDate(today.getDate() - 30); const due = new Date(today); due.setDate(today.getDate() - 15)
    const response = await fetch('/api/v1/invoices', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ clientId: clients.items[0].id, invoiceNumber, status: 'overdue', currencyCode: 'USD', amountMinor: 25000, issuedOn: issued.toISOString().slice(0, 10), dueOn: due.toISOString().slice(0, 10) }) })
    return response.status
  }, { clientName, invoiceNumber, csrfToken })
  expect(invoiceStatus).toBe(201)

  await page.getByRole('link', { name: 'Collections' }).click()
  await expect(page.getByText(/SMTP (delivery is active|is not configured)/)).toBeVisible()
  await page.getByRole('button', { name: 'Check overdue now' }).click()
  const reminderRow = page.getByRole('row').filter({ hasText: invoiceNumber })
  await expect(reminderRow.getByText(invoiceNumber, { exact: true })).toBeVisible()
  await expect(reminderRow.getByText(`billing-${suffix}@example.com`, { exact: false })).toBeVisible()
  await page.getByRole('button', { name: 'Check overdue now' }).click()
  await expect(page.getByText('0 new reminders queued')).toBeVisible()
  await reminderRow.getByRole('button', { name: 'Cancel' }).click()
  await expect(reminderRow.getByText('Cancelled', { exact: true })).toBeVisible()
})

test('approves a project contract and creates an immutable milestone invoice', async ({ page }) => {
  const suffix = Date.now()
  const projectCode = `PRJ-${suffix}`
  const contractNumber = `CTR-${suffix}`
  const invoiceNumber = `INV-PRJ-${suffix}`
  const due = new Date(); due.setDate(due.getDate() + 30)

  await signIn(page)
  await page.getByRole('link', { name: 'Projects' }).click()
  await expect(page.getByText('ARC-PORTAL', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'New project' }).click()
  await page.getByLabel('Client', { exact: true }).selectOption({ label: 'Arc & Co.' })
  await page.getByLabel('Project code').fill(projectCode)
  await page.getByLabel('Project name').fill('Controlled billing rollout')
  await page.getByLabel('Contract number').fill(contractNumber)
  await page.getByLabel('Contract title').fill('Implementation services')
  await page.getByLabel('Original contract value').fill('10000')
  await page.getByRole('button', { name: 'Create and submit' }).click()
  await expect(page.getByRole('heading', { name: projectCode })).toBeVisible()
  await expect(page.getByText('Separation of duties prevents you from reviewing your own contract.')).toBeVisible()
  await page.getByRole('button', { name: 'Close project details' }).click()
  await signOut(page)

  await signInAs(page, 'admin@ledgerops.demo', 'LedgerOpsAdmin!2026')
  await page.getByRole('link', { name: 'Projects' }).click()
  await page.getByRole('button', { name: `View ${projectCode}` }).click()
  await page.getByRole('button', { name: 'Review contract' }).click()
  await page.getByLabel('Review note').fill('Approved against the signed statement of work.')
  await page.getByRole('button', { name: 'Approve' }).click()
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible()

  await page.getByRole('button', { name: 'Add milestone' }).click()
  await page.getByLabel('Milestone name').fill('Implementation delivery')
  await page.getByLabel('Amount', { exact: true }).fill('2500')
  await page.getByLabel('Due date').fill(due.toISOString().slice(0, 10))
  await page.getByRole('button', { name: 'Add milestone' }).last().click()
  const milestone = page.locator('.project-record').filter({ hasText: 'Implementation delivery' })
  await expect(milestone).toBeVisible()
  await milestone.getByRole('button', { name: 'Invoice' }).click()
  await page.getByLabel('Invoice number').fill(invoiceNumber)
  await page.getByRole('button', { name: 'Create invoice' }).click()
  await expect(milestone.getByText(invoiceNumber, { exact: false })).toBeVisible()
  await page.getByRole('button', { name: 'Close project details' }).click()

  await page.getByRole('link', { name: 'Invoices' }).click()
  await page.getByPlaceholder('Search client or invoice…').fill(invoiceNumber)
  await expect(page.getByText(invoiceNumber, { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: `View ${invoiceNumber}` })).toBeVisible()
  await expect(page.getByRole('button', { name: `Edit ${invoiceNumber}` })).toHaveCount(0)
})

test('completes and independently approves a month-end close', async ({ page }) => {
  await signIn(page)
  await page.getByRole('link', { name: 'Month-end close' }).click()
  await expect(page.getByRole('heading', { name: 'Month-end close' })).toBeVisible()

  const nextPeriod = await page.evaluate(async () => {
    const response = await fetch('/api/v1/close-periods')
    const body = await response.json() as { items: Array<{ monthStart: string }> }
    const latest = body.items.map((item) => item.monthStart).sort().at(-1) ?? new Date().toISOString().slice(0, 7) + '-01'
    const month = new Date(`${latest}T00:00:00Z`)
    month.setUTCMonth(month.getUTCMonth() + 1)
    const due = new Date(month); due.setUTCMonth(due.getUTCMonth() + 1); due.setUTCDate(5)
    return { monthStart: month.toISOString().slice(0, 10), dueOn: due.toISOString().slice(0, 10) }
  })
  const periodLabel = new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${nextPeriod.monthStart}T00:00:00Z`))

  await page.getByRole('button', { name: 'New close period' }).click()
  await page.getByLabel('Close month').fill(nextPeriod.monthStart)
  await page.getByLabel('Close due date').fill(nextPeriod.dueOn)
  await page.getByRole('button', { name: 'Create checklist' }).click()
  await expect(page.getByText(periodLabel, { exact: true })).toBeVisible()

  for (let index = 0; index < 6; index += 1) {
    await page.getByRole('button', { name: 'Complete', exact: true }).first().click()
    await page.getByLabel('Completion evidence').fill(`Verified close control ${index + 1} against the supporting ledger.`)
    await page.getByRole('button', { name: 'Complete task' }).click()
  }
  await expect(page.getByText('Checklist complete and ready for independent review.')).toBeVisible()
  await page.getByRole('button', { name: 'Submit for review' }).click()
  await expect(page.getByText('In Review', { exact: true }).first()).toBeVisible()
  await signOut(page)

  await signInAs(page, 'admin@ledgerops.demo', 'LedgerOpsAdmin!2026')
  await page.getByRole('link', { name: 'Month-end close' }).click()
  await page.getByText(periodLabel, { exact: true }).click()
  await page.getByRole('button', { name: 'Review close' }).click()
  await page.getByLabel('Review note').fill('Verified all checklist evidence against the final close binder.')
  await page.getByRole('button', { name: 'Approve and close' }).click()
  await expect(page.getByText('Closed', { exact: true }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Review close' })).toHaveCount(0)
})

test('models a 13-week cash scenario with an append-only adjustment', async ({ page }) => {
  const suffix = Date.now()
  await signIn(page)
  await page.getByRole('link', { name: 'Cash forecast' }).click()
  await expect(page.getByRole('heading', { name: '13-week cash-flow forecast' })).toBeVisible()
  await expect(page.getByLabel('Forecast scenario', { exact: true })).toHaveValue('b1111111-1111-4111-8111-111111111111')
  await expect(page.locator('.forecast-table tbody tr')).toHaveCount(13)

  await page.getByRole('button', { name: 'Add adjustment' }).click()
  await page.getByLabel('Cash direction').selectOption('inflow')
  await page.getByLabel('Description').fill(`Owner funding ${suffix}`)
  await page.getByLabel('Amount').fill('1234')
  await page.getByRole('button', { name: 'Add adjustment', exact: true }).last().click()
  await expect(page.getByText(`Owner funding ${suffix}`, { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'New scenario' }).click()
  await page.getByLabel('Scenario name').fill(`Downside ${suffix}`)
  await page.getByLabel('Collection rate (%)').fill('60')
  await page.getByLabel('Collection delay (days)').fill('28')
  await page.getByLabel('Expense growth (%)').fill('15')
  await page.getByLabel('Milestone probability (%)').fill('40')
  await page.getByRole('button', { name: 'Create scenario' }).click()
  await expect(page.getByRole('option', { name: `Downside ${suffix}` })).toBeAttached()
  await expect(page.getByLabel('Forecast scenario', { exact: true })).toHaveValue(/.+/)
  await expect(page.getByText('60%', { exact: true })).toBeVisible()
})

test('resolves a reconciliation exception with an audit note', async ({ page }) => {
  await signIn(page)
  await page.goto('/review')
  const overdueItem = page.getByRole('button', { name: /Invoice 15 days overdue/ })
  if (await overdueItem.count()) {
    await overdueItem.click()
    await page.getByPlaceholder('What did you verify, and what action was taken?').fill('Confirmed with the client and scheduled collection follow-up.')
    await page.getByRole('button', { name: 'Mark as resolved' }).click()
  }
  await expect(page.getByText(/\d+ items? resolved/)).toBeVisible()
})
