$(_ => {
  $.fn.extend({
    withClass: function (add, clz) {
      return this.each(function () { $(this)[add ? 'addClass' : 'removeClass'](clz) })
    },
  })

  let login = '.login'
  let userfield = `${login}>.username`
  let passfield = `${login}>.password`
  let mfa = `${login}>.mfa>input`
  let cell = `${mfa}#cell`
  let email = `${mfa}#email`
  let matchable = `${login}>.matchable>input`
  let verify = `${login}>.verify`

  // let chkuser = u => u.length > 7
  let chkuser = u => u.replace(/[^0-9A-z_-]/, '') === u
    && u.length > 7
  let chkpass = p => p.length > 7
    // && p.replace(/[^,<>./?!@#$%^&*\(\)\{\}-_+=\[\]\|\\]/, '').length > 0
    // && p.replace(/[^0-9]/, '').length > 0
    && p.replace(/[^A-z]/, '').length > 0
  let chkcell = c => c.replace(/[^0-9]/, '').length == 10
  let chkmail = m => /[^@]{2,}@[^.]{2,}\.[A-z]{2,3}/.test(m)

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
        default:
          console.log('invalid arg', search)
      }
    })
    .on('logout', `>${login}`, (e, val) => {
      e.stopPropagation()

      fetch('logout', { method: 'POST' }).then(async resp => {
        if (resp.status === 202) throw {
          url: 'logout',
          method: 'POST',
          status: resp.status,
          message: await resp.text(),
        }
        $(e.currentTarget).trigger('clear')
      }).catch(ex => {
        $(document.body).trigger('error-message', ex)
      }).finally(_ => {
        console.log('finally logout, now what?')
      })
    })
    .on('forgot', `>${login}`, (e, val) => {
      e.stopPropagation()

      console.log('valid arg', 'forgot', val)
    })
    .on('reset', `>${login}`, (e, val) => {
      e.stopPropagation()

      console.log('valid arg', 'reset', val)
      $(`body>${passfield}>.change`).trigger('click')
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
    .on('change', `>${userfield}>#username`, e => {
      e.stopPropagation()

      $(e.currentTarget).trigger('test')
    })
    .on('test', `>${userfield}>#username`, e => {
      e.stopPropagation()

      let $user = $(e.currentTarget)
      let $field = $user.parent()
      let val = $user.val()
      if (!$field.withClass(chkuser(val), 'complete').hasClass('complete')) {
        $(`body>${login}`).removeAttr('id')
      } else {
        fetch(`/auth/${val}`).then(async resp => {
          if (resp.status !== 200) throw {
            url: resp.url,
            method: 'GET',
            status: resp.status,
            message: await resp.text(),
            lvl: 'console',
          }
          localStorage.username = val
          return await resp.json()
        }).then(json => $(`body>${login}`)
          .attr('id', json.id)
          .find('>.password>#password')
          .focus()
        ).catch(ex => {
          $(`body>${login}`).removeAttr('id')
          // don't refocus the username
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

      $(`body>${login}`).addClass('forgetting')
    })

    // password events
    .on('keyup', `>${passfield}>#password`, e => {
      e.stopPropagation()

      let $pass = $(e.currentTarget)
      $pass.parent().withClass(chkpass($pass.val()), 'complete')
    })
    .on('click', `>[id]${passfield}>.ok`, e => {
      e.stopPropagation()

      let body = {
        id: $(`body>${login}`).attr('id'),
        password: $(`body>${passfield}>#password`).val(),
      }

      fetch('auth', { method: 'POST', body: JSON.stringify(body) })
        .then(async resp => {
          // // FIXME: any self-respecting login returns a location
          // if (resp.status !== 301) throw {
          if (resp.status !== 200) throw {
            url: resp.url,
            method: 'POST',
            status: resp.status,
            message: await resp.text(),
          }
          // console.log('headers', resp.headers)
          return resp.headers['Location']
        })
        // FIXME: null location should be an exception
        .then(location => document.location = location ?? '/')
        .catch(ex => {
          $(`body>${password}`).focus()
          $(document.body).trigger('error-message', ex)
        })
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
        id: $(`body>${login}>`).attr('id'),
        email: $(`body>${email}`).val() || null,
        cell: $(`body>${cell}`).val() || null,
      }

      fetch('auth', { method: 'DELETE', body: JSON.stringify(body) })
        .then(async resp => {
          if (resp.status !== 200) throw {
            url: resp.url,
            method: 'POST',
            status: resp.status,
            message: await resp.text(),
          }
          // what do we do here? static text explaining to check email/sms
          // for a link to reset? something else?
          $(`body>${login}>.forgot.cancel`).trigger('click')
        })
        .catch(ex => {
          $(document.body).trigger('error-message', ex)
        })
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
        && $(`>${passfield}>#password`).val() === val
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
        username: $(`body>${userfield}>#username`).val(),
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
          return await resp.text()
        })
        .then(id => $(`body>${login}`).attr('id', id).removeClass('adding'))
        .then(_ => setTimeout($(e.currentTarget).click(), 150))
        .catch(ex => $(document.body).trigger('error-message', ex))
    })

    // changing password
    .on('click', `>[id]:not(.mismatch)${verify}>.save`, e => {
      e.stopPropagation()

      let id = $(`body>${login}`).attr('id')
      let body = {
        old: { id, password: $(`body>${passfield}>#password`).val() },
        new: { id, password: $(`body>${verify}>#verify`).val() },
      }

      fetch(`/auth/${$dt.attr('id')}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
      }).then(async resp => {
        // // FIXME: PATCH also needs to return redirect, like POST above
        // if (resp.status !== 301) throw {
        if (resp.status !== 204) throw {
          url: resp.url,
          method: 'POST',
          status: resp.status,
          message: await resp.text(),
        }
        // console.log('headers', resp.headers)
        return resp.headers['Location']
      }).then(location =>
        // FIXME: null location should be an exception
        document.location = location ?? '/'
      ).catch(ex => {
        $(document.body).trigger('error-message', ex)
      })
    })

    // grand-unified cancel for editing and adding
    .on('click', `>${verify}>.cancel`, e => {
      e.stopPropagation()

      $(`body>${login}`).removeClass('editing adding')
    })

    // error handling
    .on('error-message', (e, ex) => {
      e.stopPropagation()

      if (ex.lvl === 'console')
        console.error('error-message', ex)
      else $(e.currentTarget)
        .addClass('messaging')
        .find('>.message')
        .trigger('send', ex)
    })
    .on('send', '>.message', (e, ex) => {
      e.stopPropagation()

      let $msg = $(e.currentTarget).addClass(ex.lvl ?? 'error')

      let $exception = $msg.find('>.window>.exception')
      Object.entries(ex).forEach(([k, v]) => {
        $exception.find(`>.${k}`).text(v)
      })

      e.currentTarget.timer = setTimeout(_ => $msg
        .find('>.window>.ok')
        .click(),
        (ex.to ?? 30) * 1000)
    })
    .on('click', '>.message>.window>.ok', (e, ex) => {
      e.stopPropagation()

      clearTimeout($(document.body)
        .removeClass('messaging')
        .find('>.message')
        .get(0)
        .timer)
    })

    // start with a clean slate
    .find(`>${login}`)
    .trigger('init')
})
