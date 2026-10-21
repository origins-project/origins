import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Chart from '../ui/Chart.vue'
import Lightbox from '../ui/Lightbox.vue'

function pointer(type: string, init: PointerEventInit): PointerEvent {
  return new PointerEvent(type, { bubbles: true, pointerId: 1, button: 0, buttons: 1, ...init })
}

describe('a chart', () => {
  it('stops panning once the button is up, wherever it was let go', async () => {
    const wrapper = mount(Chart, {
      props: { boxes: [], lines: [], width: 100, height: 100, box: { width: 10, height: 10 }, label: 'chart' },
    })
    const frame = wrapper.find('.frame').element
    frame.setPointerCapture = () => {}
    const transform = () => wrapper.find('svg > g').attributes('transform')
    const before = transform()
    frame.dispatchEvent(pointer('pointerdown', { clientX: 10, clientY: 10 }))
    // let go outside the frame, so that it never hears the pointer come up
    frame.dispatchEvent(pointer('pointermove', { clientX: 60, clientY: 60, buttons: 0 }))
    await nextTick()
    expect(transform()).toBe(before)
  })
})

describe('a lightbox', () => {
  it('keeps a box drawn when the pointer comes up, and not when it is cancelled', async () => {
    const wrapper = mount(Lightbox, {
      props: { open: true, title: 'picture', src: 'a.jpg', draw: true },
      attachTo: document.body,
    })
    await nextTick()
    const picture = document.querySelector<HTMLElement>('[data-picture]')!
    picture.getBoundingClientRect = () => ({ left: 0, top: 0, width: 100, height: 100 }) as DOMRect
    const drag = (end: string) => {
      picture.dispatchEvent(pointer('pointerdown', { clientX: 10, clientY: 10 }))
      picture.dispatchEvent(pointer('pointermove', { clientX: 50, clientY: 60 }))
      picture.dispatchEvent(pointer(end, { clientX: 50, clientY: 60 }))
    }

    drag('pointercancel')
    expect(wrapper.emitted('update:drawn')).toBeUndefined()
    drag('pointerup')
    expect(wrapper.emitted('update:drawn')).toEqual([[[0.1, 0.1, 0.5, 0.6]]])
    wrapper.unmount()
  })
})
