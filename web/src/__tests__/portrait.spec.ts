import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Portrait from '../ui/Portrait.vue'

// loaded tells the portrait its picture arrived at the given size.
async function loaded(wrapper: ReturnType<typeof mount>, width: number, height: number) {
  const img = wrapper.find('img')
  Object.defineProperty(img.element, 'naturalWidth', { value: width })
  Object.defineProperty(img.element, 'naturalHeight', { value: height })
  await img.trigger('load')
}

describe('a portrait', () => {
  it('is hidden until its picture is loaded, then cropped to the region', async () => {
    const wrapper = mount(Portrait, { props: { src: 'a.jpg', region: [0.25, 0.25, 0.75, 0.75] } })
    expect(wrapper.classes()).toContain('waiting')
    await loaded(wrapper, 400, 400)
    expect(wrapper.classes()).not.toContain('waiting')
    expect(wrapper.classes()).toContain('cropped')
  })

  it('shows the whole picture where its region cannot be cropped to', async () => {
    for (const region of [[0.5, 0.2, 0.5, 0.8], [0.1, 0.2, 0.3]]) {
      const wrapper = mount(Portrait, { props: { src: 'a.jpg', region } })
      await loaded(wrapper, 400, 300)
      expect(wrapper.classes()).not.toContain('waiting')
      expect(wrapper.classes()).not.toContain('cropped')
    }
  })

  it('is marked where a draft changes it, as a value is', () => {
    expect(mount(Portrait, { props: { as: 'button', marked: true } }).attributes()).toHaveProperty('data-drafted')
    expect(mount(Portrait, { props: { as: 'button' } }).attributes()).not.toHaveProperty('data-drafted')
  })
})
