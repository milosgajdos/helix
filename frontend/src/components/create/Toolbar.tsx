import React, { FC } from 'react'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import ConstructionIcon from '@mui/icons-material/Construction'
import LoginIcon from '@mui/icons-material/Login'

import Cell from '../widgets/Cell'
import Row from '../widgets/Row'
import SessionModeSwitch from './SessionModeSwitch'
import SessionModeDropdown from './SessionModeDropdown'
import ModelPicker from './ModelPicker'

import useIsBigScreen from '../../hooks/useIsBigScreen'
import useAccount from '../../hooks/useAccount'

import {
  ISessionMode,
  ISessionType,
  IApp,
  SESSION_MODE_INFERENCE,
  SESSION_TYPE_TEXT,
} from '../../types'

const CreateToolbar: FC<{
  mode: ISessionMode,
  type: ISessionType,
  model?: string,
  app?: IApp,
  onOpenConfig: () => void,
  onSetMode: (mode: ISessionMode) => void,
  onSetModel: (model: string) => void,
}> = ({
  mode,
  type,
  model,
  app,
  onOpenConfig,
  onSetMode,
  onSetModel,
}) => {
  const bigScreen = useIsBigScreen()
  const account = useAccount()
  const appRequested = new URLSearchParams(window.location.search).get('app_id') || '';
  return (
    <Row>
      <Cell>
        {
          !(app || appRequested) && mode === SESSION_MODE_INFERENCE && type === SESSION_TYPE_TEXT && (
            <ModelPicker
              model={model || ''}
              onSetModel={onSetModel}
            />
          )
        }
      </Cell>
      <Cell grow>
        
      </Cell>
      {
        // don't show the tools icon in inference mode since we don't have
        // global tools any more. we still show it in "learn" mode where it
        // controls rag and finetune settings.
        !app && !(mode === SESSION_MODE_INFERENCE) && (
          <Cell>
            <IconButton
              onClick={ onOpenConfig }
            >
              <ConstructionIcon />
            </IconButton>
          </Cell>
        )
      }
      {
        !app && (
          <Cell>
            {
              bigScreen ? (
                <SessionModeSwitch
                  mode={ mode }
                  onSetMode={ onSetMode }
                />
              ) : (
                <SessionModeDropdown
                  mode={ mode }
                  onSetMode={ onSetMode }
                />
              )
            }
          </Cell>
        )
      }
      <Cell>
        {
          !account.user && (bigScreen ? (
            <Button
              id='login-button'
              size="medium"
              variant="contained"
              color="primary"
              endIcon={ <LoginIcon /> }
              onClick={ account.onLogin }
              sx={{
                ml: 2,
              }}
            >
              Login / Register
            </Button> 
          ) : (
            <IconButton
              onClick={ () => account.onLogin() }
            >
              <LoginIcon />
            </IconButton>
          ))
        }
      </Cell>
    </Row>
  )
}

export default CreateToolbar
