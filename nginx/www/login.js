$(_ => {
  $.fn.extend({
    withClass: function (add, clz) {
      return this.each(function () { $(this)[add ? 'addClass' : 'removeClass'](clz) })
    },
  })

  let login = '.login'
  let userfield = `${login}>.username`
  let username = `${userfield}>#username`
  let passfield = `${login}>.password`
  let password = `${passfield}>#password`
  let mfa = `${login}>.mfa>input`
  let cell = `${mfa}#cell`
  let email = `${mfa}#email`
  let matchable = `${login}>.matchable>input`
  let verify = `${login}>.verify`

  // let chkuser = u => u.length > 7
  let chkuser = u => u.replace(/[^0-9A-Za-z_-]/, '') === u
    && u.length > 7
  let chkpass = p => p.length > 7
    // && p.replace(/[^,<>./?!@#$%^&*\(\)\{\}-_+=\[\]\|\\]/, '').length > 0
    // && p.replace(/[^0-9]/, '').length > 0
    && p.replace(/[^A-Za-z]/, '').length > 0
  let chkcell = c => c.replace(/[^0-9]/, '').length == 10
  let chkmail = m => /[^@]{2,}@[^.]{2,}\.[A-Za-z]{2,3}/.test(m)
  let keyignore = Object.fromEntries([
    // it lacks the efficiency of golang maps, but it's potentially 
    // cleaner than list.indexOf(...)
    'ArrowDown',
    'ArrowLeft',
    'ArrowRight',
    'ArrowUp',
    'Alt',
    'Control',
    'End',
    'Escape',
    'Home',
    'Meta',
    'PageDown',
    'PageUp',
    'Shift',
    'Tab',
  ].map(v => [v, 1]))

  $(document.body)
    // form maintenance
    .on('init', `>${login}`, e => {
      e.stopPropagation()

      $(e.currentTarget)
        .trigger('clear')
        .find('>.username>#username')
        .val(localStorage.username)
        .trigger('test')

      let search = document.location.search
        .replace(/^\?/, '')
        .split('&')
        .filter(v => v)
        // .map(v => v.split(/=/, 2))
        .reverse()
        .pop()

      switch (search) {
        case 'logout':
        case 'reset':
        case 'forgot':
          // $(e.currentTarget).trigger(...search)
          $(e.currentTarget).trigger(search)
          break
        case undefined: break
        default: $(document.body).trigger('error-message', {
          message: `unknown query param: '${search}'`,
          level: 'console',
        })
      }
    })
    .on('logout', `>${login}`, (e, val) => {
      e.stopPropagation()

      fetch('logout', { method: 'POST' })
        .then(async resp => {
          let result = {
            url: 'logout',
            method: 'POST',
            status: resp.status,
          }
          if (resp.status === 202) throw {
            ...result,
            message: await resp.text(),
          }

          $(e.currentTarget).trigger('clear')
          return {
            ...result,
            level: 'info',
            timeout: 5,
            redirect: _ => $(`body>${username}`)
              .val(localStorage.username)
              .trigger('change'),
          }
        })
        .then(done => $(document.body).trigger('error-message', done))
        .catch(ex => $(document.body).trigger('error-message', ex))
    })
    .on('forgot', `>${login}`, (e, val) => {
      e.stopPropagation()

      $(`body>${login}`).addClass('forgetting')
    })
    .on('reset', `>${login}`, (e, val) => {
      e.stopPropagation()

      // might still have to enter username yourself if it's not on
      // localStorage, but we're not sending it; chances are `GET /auth`
      // hasn't returned yet so we can't use the button click
      $(`body>${login}`)
        .addClass('editing')
        .trigger('password-mgmt')
        .find(`.password, .username>.forgot`)
        .hide()
    })
    .on('clear', `>${login}`, e => {
      e.stopPropagation()

      $(e.currentTarget)
        .removeClass('editing adding mismatch forgetting')
        .removeAttr('id')
        .find('input')
        .val('')
        .trigger('change')
        .trigger('keyup') // XXX: one or the other
    })
    .on('password-mgmt', `>${login}`, e => {
      e.stopPropagation()

      // sets the mismatch flag to hide the ok button
      $(`body>${matchable}`)
        .val('')
        .first()
        .trigger('keyup')
    })

    // username events
    .on('change', `>${username}`, e => {
      e.stopPropagation()

      $(e.currentTarget).trigger('test')
    })
    .on('keyup', `>${username}`, e => {
      e.stopPropagation()

      if (keyignore[e.key]) {
        return
        // } else {
        //   console.log(e.key)
      }

      let input = e.currentTarget

      clearTimeout(input.timer ?? -1)

      // // XXX: hold off on this
      // $(`body>${login}>.state`).text(input.value)

      input.timer = setTimeout(_ => $(input).trigger('test'), 100)
    })
    .on('test', `>${username}`, e => {
      e.stopPropagation()

      let input = e.currentTarget

      clearTimeout(input.timer ?? -1)
      delete input.timer

      let val = input.value
      if (!$(input.parentNode).withClass(chkuser(val), 'complete').hasClass('complete')) {
        $(`body>${login}`).removeAttr('id')
      } else {
        fetch(`/auth/${val}`).then(async resp => {
          if (resp.status !== 200) throw {
            url: resp.url,
            method: 'GET',
            status: resp.status,
            message: await resp.text(),
            level: 'console',
          }
          localStorage.username = val
          return await resp.json()
        }).then(json => $(`body>${login}`).attr('id', json.id)
        ).catch(ex => {
          $(`body>${login}`).removeAttr('id')
          $(document.body).trigger('error-message', ex)
        })
      }
    })
    .on('click', `>${userfield}>.create`, e => {
      e.stopPropagation()

      $(`body>${login}`)
        .addClass('adding')
        .trigger('password-mgmt')
        .find('>.password>#password')
        .val('') // needs to be clear for the PATCH part of account creation
        .trigger('keyup')
    })
    .on('click', `>[id]${userfield}>.forgot`, e => {
      e.stopPropagation()

      $(`body>${login}`).trigger('forgot')
    })

    // password events
    .on('keyup', `>${password}`, e => {
      e.stopPropagation()

      let $pass = $(e.currentTarget)
      $pass.parent().withClass(chkpass($pass.val()), 'complete')
    })
    .on('click', `>[id]${passfield}>.ok`, e => {
      e.stopPropagation()

      let body = {
        id: $(`body>${login}`).attr('id'),
        password: $(`body>${password}`).val(),
      }

      fetch('auth', { method: 'POST', body: JSON.stringify(body) })
        .then(async resp => {
          if (resp.status !== 200) throw {
            url: resp.url,
            method: 'POST',
            status: resp.status,
            message: await resp.text(),
          }
          return resp.url
        })
        .then(redirect => document.location = redirect)
        .catch(ex => $(document.body).trigger('error-message', {
          ...ex,
          redirect: _ => $(`body>${password}`).focus()
        }))
    })
    .on('click', `>[id]${passfield}>.change`, e => {
      e.stopPropagation()

      $(`body>${login}`)
        .addClass('editing')
        .trigger('password-mgmt')
    })

    // forgot password
    .on('click', `>[id]:not(.missing)${login}>.forgot.reset`, e => {
      e.stopPropagation()

      let body = {
        id: $(`body>${login}`).attr('id'),
        email: $(`body>${email}`).val() || null,
        cell: $(`body>${cell}`).val() || null,
        redirect: `${document.location.pathname}?reset`,
      }

      fetch('auth', { method: 'DELETE', body: JSON.stringify(body) })
        .then(async resp => {
          let result = {
            url: resp.url,
            method: 'DELETE',
            status: resp.status,
            redirect: _ => $(`body>${login}>.forgot.cancel`).trigger('click'),
          }

          if (resp.status !== 204) throw {
            ...result,
            level: 'error',
            message: await resp.text(),
          }

          return {
            ...result,
            level: 'info',
            message: 'password has been reset; follow the link sent to your email/text to complete the reset',
          }
        })
        .then(success => $(document.body).trigger('error-message', success))
        .catch(ex => $(document.body).trigger('error-message', ex))
    })
    .on('click', `>[id]${login}>.forgot.cancel`, e => {
      e.stopPropagation()

      $(`body>${login}`).removeClass('forgetting')
    })

    // validate mfa fields
    .on('keyup', `>${cell}`, e => {
      e.stopPropagation()

      let val = $(e.currentTarget).val()
      $(e.currentTarget.parentNode)
        .withClass(!val, 'empty')
        .withClass(chkcell(val), 'complete')
    })
    .on('keyup', `>${email}`, e => {
      e.stopPropagation()

      let val = $(e.currentTarget).val()
      $(e.currentTarget.parentNode)
        .withClass(!val, 'empty')
        .withClass(chkmail(val), 'complete')
    })
    .on('keyup', `>${mfa}`, e => {
      e.stopPropagation()

      $(`body>${login}`).withClass($(`body>${mfa}`)
        .filter((_, v) => /\bcomplete\b/.test(v.parentNode.className))
        .length === 0,
        'missing')
    })

    // confirm new password
    .on('keyup', `>${matchable}`, e => {
      e.stopPropagation()

      let val = $(e.currentTarget).val()

      if (!chkpass(val)) {
        $(`body>${login}`).addClass('mismatch')
      } else if ($(`body>${matchable}`)
        .filter((_, v) => v.value !== val) // hamfisted, but it's only 2 fields
        .length
      ) {
        $(`body>${login}`).addClass('mismatch')
      } else if ($(`body>${login}`).hasClass('editing')
        && $(`>${password}`).val() === val
      ) {
        $(document.body).trigger('error-message', {
          message: 'new and old passwords are the same'
        })
        $(`body>${login}`).addClass('mismatch')
      } else {
        $(`body>${login}`).removeClass('mismatch')
      }
    })

    // creating user
    .on('click', `>.adding:not(.mismatch)${verify}>.save`, e => {
      e.stopPropagation()

      let body = {
        username: $(`body>${username}`).val(),
        email: $(`body>${email}`).val() || null,
        cell: $(`body>${cell}`).val() || null,
      }

      fetch('user', { method: 'POST', body: JSON.stringify(body) })
        .then(async resp => {
          if (resp.status !== 201) throw {
            url: resp.url,
            method: 'POST',
            status: resp.status,
            message: await resp.text(),
          }
          localStorage.username = body.username
          return await resp.text()
        })
        .then(id => $(`body>${login}`).attr('id', id).removeClass('adding'))
        .then(_ => setTimeout(_ => $(e.currentTarget).click(), 500))
        .catch(ex => $(document.body).trigger('error-message', ex))
    })

    // changing password
    .on('click', `>[id]:not(.mismatch)${verify}>.save`, e => {
      e.stopPropagation()

      let id = $(`body>${login}`).attr('id')
      let body = {
        old: $(`body>${password}`).val(),
        new: $(`body>${verify}>#verify`).val(),
      }

      if (body.old == body.new) { // the service would do this anyway
        return $(document.body).trigger('error-message', {
          url: `/auth/${id}`,
          method: 'PATCH',
          status: 400,
          message: 'passwords match'
        })
      }

      fetch(`/auth/${id}`, { method: 'PATCH', body: JSON.stringify(body) })
        .then(async resp => {
          let result = {
            url: resp.url,
            method: 'PATCH',
            status: resp.status,
          }

          if (resp.status !== 204) throw {
            ...result,
            message: await resp.text(),
          }

          return {
            ...result,
            message: 'password changed',
            level: 'info',
            timeout: 5,
            redirect: _ => document.location = resp.headers.get('Location'),
          }
        })
        .then(result => $(document.body).trigger('error-message', result))
        .catch(ex => $(document.body).trigger('error-message', ex))
    })

    // grand-unified cancel for editing and adding
    .on('click', `>${verify}>.cancel`, e => {
      e.stopPropagation()

      $(`body>${login}`).removeClass('editing adding')
    })

    // error handling
    .on('error-message', (e, ex) => {
      e.stopPropagation()

      let log = ex.level === 'info' ? console.log : console.error
      log('error-message', ex)

      if (ex.level === 'console') {
        return
      }

      $(e.currentTarget)
        .addClass('messaging')
        .find('>.message')
        .trigger('send', ex)
    })
    .on('send', '>.message', (e, ex) => {
      e.stopPropagation()

      let {
        level,
        timeout,
        redirect,
        ...fields
      } = ex

      let $msg = $(e.currentTarget).addClass(level ?? 'error')

      let $exception = $msg.find('>.window>.exception')
      Object.entries(fields).forEach(([k, v]) => {
        $exception.find(`>.${k}`).text(v)
      })

      e.currentTarget.dismissed = redirect ?? (_ => _)
      e.currentTarget.timer = setTimeout(_ => $msg
        .find('>.window>.ok')
        .click(),
        (timeout ?? 30) * 1000)
    })
    .on('click', '>.message>.window>.ok', e => {
      e.stopPropagation()

      let message = $(document.body)
        .removeClass('messaging')
        .find('>.message')
        .get(0)

      clearTimeout(message.timer)

      message.dismissed()
    })

    // start with a clean slate
    .find(`>${login}`)
    .trigger('init')
})
