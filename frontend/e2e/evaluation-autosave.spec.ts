import { expect, test, type Page, type Route } from '@playwright/test'

interface ScoreServer {
  score: number | null
  revision: number
  receivedBaseRevisions: number[]
}

const challengeId = '11111111-1111-4111-8111-111111111111'
const contestId = '22222222-2222-4222-8222-222222222222'
const performanceId = '33333333-3333-4333-8333-333333333333'
const contestantId = '44444444-4444-4444-8444-444444444444'
const juryId = '55555555-5555-4555-8555-555555555555'
const criterionId = '66666666-6666-4666-8666-666666666666'

function envelope(data: unknown) {
  return JSON.stringify({ data, request_id: 'e2e' })
}

async function mockEvaluationAPI(page: Page, server: ScoreServer) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname.replace('/api/v1', '')

    if (path === '/auth/refresh') {
      return json(route, 200, { access_token: 'e2e-token', expires_at: new Date(Date.now() + 60_000).toISOString() })
    }
    if (path === '/auth/me') {
      return json(route, 200, {
        id: juryId,
        login: 'jury-e2e',
        full_name: 'Тестовое жюри',
        roles: ['JURY'],
        must_change_password: false,
      })
    }
    if (path === '/config') {
      return json(route, 200, {
        app_name: 'e2e',
        env: 'test',
        features: {
          reference_cms: true,
          email_notifications: false,
          participant_cabinet: true,
          attendance: true,
          points: true,
          merch: true,
          predictions: false,
          jury: true,
        },
      })
    }
    if (path === `/jury/challenges/${challengeId}/live`) {
      return json(route, 200, liveSnapshot())
    }
    if (path === `/jury/challenges/${challengeId}/scorecard` && request.method() === 'GET') {
      return json(route, 200, scorecard(server))
    }
    if (path === `/jury/challenges/${challengeId}/scorecard` && request.method() === 'PUT') {
      const body = request.postDataJSON() as {
        criterion_id: string
        score: number
        base_revision: number
      }
      server.receivedBaseRevisions.push(body.base_revision)
      if (body.base_revision !== server.revision) {
        return route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            error: {
              code: 'EVALUATION_REVISION_CONFLICT',
              message: 'conflict',
              details: { current_score: server.score, current_revision: server.revision },
            },
            request_id: 'e2e',
          }),
        })
      }
      server.score = body.score
      server.revision++
      return json(route, 200, {
        criterion_id: body.criterion_id,
        score: server.score,
        revision: server.revision,
        total: server.score,
      })
    }
    return route.fulfill({ status: 404, contentType: 'application/json', body: envelope(null) })
  })
}

async function json(route: Route, status: number, data: unknown) {
  return route.fulfill({ status, contentType: 'application/json', body: envelope(data) })
}

function liveSnapshot() {
  return {
    challenge_id: challengeId,
    contest_id: contestId,
    challenge_title: 'Автопортрет E2E',
    session_revision: 1,
    state: 'LIVE',
    current_contestant_user_id: contestantId,
    current_performance_id: performanceId,
    current_phase_id: null,
    started_at: new Date().toISOString(),
    finished_at: null,
    phase_started_at: new Date().toISOString(),
    phase_duration_seconds: 300,
    paused_at: null,
    accumulated_pause_seconds: 0,
    timer_remaining_seconds: 240,
    jury_online: 1,
    server_time: new Date().toISOString(),
    current: { user_id: contestantId, full_name: 'Конкурсант E2E', organization: 'Тест' },
    performance: {
      id: performanceId,
      contestant_user_id: contestantId,
      status: 'ACTIVE',
      sequence_number: 1,
      started_at: new Date().toISOString(),
      finished_at: null,
    },
    phases: [],
    contestants: [{ user_id: contestantId, full_name: 'Конкурсант E2E', organization: 'Тест' }],
    drawn: true,
    draw_locked: true,
    scheme_type: 'CRITERIA_SCORING',
  }
}

function scorecard(server: ScoreServer) {
  return {
    configured: true,
    scheme_type: 'CRITERIA_SCORING',
    scoring_ui: 'CRITERIA',
    editable: true,
    performance_id: performanceId,
    contestant: { user_id: contestantId, full_name: 'Конкурсант E2E', organization: 'Тест' },
    criteria: [
      {
        id: criterionId,
        title: 'Содержание',
        description: null,
        min_score: 1,
        max_score: 10,
        weight: 1,
        is_required: true,
        sort_order: 0,
        bands: [],
        score: server.score,
        revision: server.revision,
      },
    ],
    filled: server.score == null ? 0 : 1,
    total: server.score,
  }
}

async function pendingCount(page: Page): Promise<number> {
  return page.evaluate(async () => {
    const db = await new Promise<IDBDatabase>((resolve, reject) => {
      const request = indexedDB.open('student-leader-evaluation', 1)
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })
    return new Promise<number>((resolve, reject) => {
      const request = db.transaction('pending_scores', 'readonly').objectStore('pending_scores').count()
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })
  })
}

test('keeps an offline score in IndexedDB, syncs after reconnect, and survives refresh', async ({ page, context }) => {
  const server: ScoreServer = { score: null, revision: 0, receivedBaseRevisions: [] }
  await mockEvaluationAPI(page, server)
  await page.goto(`/jury/challenges/${challengeId}`)
  await expect(page.getByRole('heading', { name: 'Автопортрет E2E' })).toBeVisible()

  await context.setOffline(true)
  await page.getByRole('button', { name: '8', exact: true }).click()
  await expect(page.getByText(/Нет соединения.*на устройстве: 1/)).toBeVisible()
  await expect.poll(() => pendingCount(page)).toBe(1)

  await context.setOffline(false)
  await expect(page.getByText('Все изменения сохранены')).toBeVisible({ timeout: 10_000 })
  await expect.poll(() => pendingCount(page)).toBe(0)
  expect(server.score).toBe(8)

  await page.reload()
  await expect(page.getByRole('button', { name: '8', exact: true })).toHaveAttribute('aria-pressed', 'true')
})

test('rebases an unsynced local value on a newer server revision without losing it', async ({ page }) => {
  const server: ScoreServer = { score: 7, revision: 1, receivedBaseRevisions: [] }
  await mockEvaluationAPI(page, server)
  await page.goto(`/jury/challenges/${challengeId}`)
  await expect(page.getByRole('button', { name: '7', exact: true })).toHaveAttribute('aria-pressed', 'true')

  server.score = 8
  server.revision = 2
  await page.getByRole('button', { name: '9', exact: true }).click()

  await expect(page.getByText('Все изменения сохранены')).toBeVisible({ timeout: 10_000 })
  expect(server.receivedBaseRevisions).toEqual([1, 2])
  expect(server.score).toBe(9)
  expect(server.revision).toBe(3)
  await expect(page.getByRole('button', { name: '9', exact: true })).toHaveAttribute('aria-pressed', 'true')
})
