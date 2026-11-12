import { defineComponent, h } from 'vue'
import { siMeta, siGoogleads, siWhatsapp } from 'simple-icons'
import { Brain } from 'lucide-vue-next'

export const MetaAdsIcon = defineComponent({
  name: 'MetaAdsIcon',
  props: {
    size: {
      type: [Number, String],
      default: 15,
    },
  },
  setup(props) {
    return () =>
      h(
        'svg',
        {
          xmlns: 'http://www.w3.org/2000/svg',
          viewBox: '0 0 24 24',
          width: props.size,
          height: props.size,
          fill: 'currentColor',
          'aria-hidden': 'true',
        },
        [h('path', { d: siMeta.path })],
      )
  },
})

export const GoogleAdsIcon = defineComponent({
  name: 'GoogleAdsIcon',
  props: {
    size: {
      type: [Number, String],
      default: 15,
    },
  },
  setup(props) {
    return () =>
      h(
        'svg',
        {
          xmlns: 'http://www.w3.org/2000/svg',
          viewBox: '0 0 24 24',
          width: props.size,
          height: props.size,
          fill: 'currentColor',
          'aria-hidden': 'true',
        },
        [h('path', { d: siGoogleads.path })],
      )
  },
})

export const WhatsAppIcon = defineComponent({
  name: 'WhatsAppIcon',
  props: {
    size: {
      type: [Number, String],
      default: 15,
    },
  },
  setup(props) {
    return () =>
      h(
        'svg',
        {
          xmlns: 'http://www.w3.org/2000/svg',
          viewBox: '0 0 24 24',
          width: props.size,
          height: props.size,
          fill: 'currentColor',
          'aria-hidden': 'true',
        },
        [h('path', { d: siWhatsapp.path })],
      )
  },
})

export const PixelBrainIcon = defineComponent({
  name: 'PixelBrainIcon',
  props: {
    size: {
      type: [Number, String],
      default: 15,
    },
  },
  setup(props) {
    return () =>
      h(Brain, {
        size: Number(props.size) || 15,
        'stroke-width': 2,
        'aria-hidden': 'true',
      })
  },
})

export const BrainIcon = PixelBrainIcon
