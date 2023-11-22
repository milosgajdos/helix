import React, { FC } from 'react'
import { DialogProps } from '@mui/material/Dialog'
import {CopyToClipboard} from 'react-copy-to-clipboard'
import Button from '@mui/material/Button'
import Window from './Window'
import JsonView from './JsonView'
import useSnackbar from '../../hooks/useSnackbar'

interface JsonWindowProps {
  data: any,
  copyToClipboard?: boolean,
  size?: DialogProps["maxWidth"],
  onClose: {
    (): void,
  },
}

const JsonWindow: FC<React.PropsWithChildren<JsonWindowProps>> = ({
  data,
  copyToClipboard = true,
  size = 'md',
  onClose,
}) => {
  const snackbar = useSnackbar()
  return (
    <Window
      open
      withCancel
      size={ size }
      cancelTitle="Close"
      onCancel={ onClose }
      leftButtons={(
        <CopyToClipboard
          text={JSON.stringify(data, null, 4)}
          onCopy={ () => {
            snackbar.success('Copied to clipboard')
          }}
        >
          <Button
            color="secondary"
            variant="contained"
          >
            Copy to clipboard
          </Button>
        </CopyToClipboard>
      )}
    >
      <JsonView
        data={ data }
        scrolling={ false }
      />
    </Window>
  )
}

export default JsonWindow