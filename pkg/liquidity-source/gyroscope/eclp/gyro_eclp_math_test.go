package gyroeclp

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/stretchr/testify/assert"
)

func Test_calcAChiAChiInXp(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("2153421421514125231432")
			beta   = int256.MustFromDec("351242142152144352142315")
			c      = int256.MustFromDec("4542142342121424214")
			s      = int256.MustFromDec("552142314423142324214")
			lambda = int256.MustFromDec("532142342142142144365")

			tauAlphaX = int256.MustFromDec("152142154332623423")
			tauAlphaY = int256.MustFromDec("55353432140214225325")
			tauBetaX  = int256.MustFromDec("215432632543253245215")
			tauBetaY  = int256.MustFromDec("6421521532542534246431")
			u         = int256.MustFromDec("1153452543263253253524")
			v         = int256.MustFromDec("326533539257394275394645")
			w         = int256.MustFromDec("269304283058430853402583532")
			z         = int256.MustFromDec("64430286340583402583905215")
			dSq       = int256.MustFromDec("20000000000000000000000000000000000000000")

			// [2153421421514125231432, 351242142152144352142315, 4542142342121424214, 552142314423142324214, 532142342142142144365]
			// [[152142154332623423, 55353432140214225325], [215432632543253245215, 6421521532542534246431], 1153452543263253253524, 326533539257394275394645, 269304283058430853402583532, 64430286340583402583905215, 20000000000000000000000000000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "7519295519963"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("21534214423421514125231432")
			beta   = int256.MustFromDec("35151242142152144352142315")
			c      = int256.MustFromDec("454214514512342121424214")
			s      = int256.MustFromDec("552142314423345235142324214")
			lambda = int256.MustFromDec("53214234221424142142144365")

			tauAlphaX = int256.MustFromDec("1521421553254332623423")
			tauAlphaY = int256.MustFromDec("55352143432140214225325")
			tauBetaX  = int256.MustFromDec("21543265234532543253245215")
			tauBetaY  = int256.MustFromDec("642152153253452542534246431")
			u         = int256.MustFromDec("115363465452543263253253524")
			v         = int256.MustFromDec("3265334314539257394275394645")
			w         = int256.MustFromDec("269304283214058430853402583532")
			z         = int256.MustFromDec("644304231286340583402583905215")
			dSq       = int256.MustFromDec("50000000000000000000000000000000000000000")

			// [21534214423421514125231432, 35151242142152144352142315, 454214514512342121424214, 552142314423345235142324214, 53214234221424142142144365]
			// [[1521421553254332623423, 55352143432140214225325], [21543265234532543253245215, 642152153253452542534246431], 115363465452543263253253524, 3265334314539257394275394645, 269304283214058430853402583532, 644304231286340583402583905215, 50000000000000000000000000000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "-106317595963176963860294275980"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("51421534214423421514125231432")
			beta   = int256.MustFromDec("35151251342142152144352142315")
			c      = int256.MustFromDec("4542145145123426235121424214")
			s      = int256.MustFromDec("55214231451423345235142324214")
			lambda = int256.MustFromDec("5321423422142414623532142144365")

			tauAlphaX = int256.MustFromDec("1521421553254332614223423")
			tauAlphaY = int256.MustFromDec("5535253426143432140214225325")
			tauBetaX  = int256.MustFromDec("2154326562345234532543253245215")
			tauBetaY  = int256.MustFromDec("64215215325345234552542534246431")
			u         = int256.MustFromDec("115363465452543623263253253524")
			v         = int256.MustFromDec("32653343146543539257394275394645")
			w         = int256.MustFromDec("2693042832140325558430853402583532")
			z         = int256.MustFromDec("644304231286340583532402583905215")
			dSq       = int256.MustFromDec("90000000000000000000000000000000000000000")

			// [51421534214423421514125231432, 35151251342142152144352142315, 4542145145123426235121424214, 55214231451423345235142324214, 5321423422142414623532142144365]
			// [[1521421553254332614223423, 5535253426143432140214225325], [2154326562345234532543253245215, 64215215325345234552542534246431], 115363465452543623263253253524, 32653343146543539257394275394645, 2693042832140325558430853402583532, 644304231286340583532402583905215, 90000000000000000000000000000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "-1394710676864973574300922114887120875638745670"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("514215514125231432")
			beta   = int256.MustFromDec("35151251342142152144352142315")
			c      = int256.MustFromDec("4542145235121424214")
			s      = int256.MustFromDec("55214231451423345235142324214")
			lambda = int256.MustFromDec("53214232142144365")

			tauAlphaX = int256.MustFromDec("1521421553254332614223423")
			tauAlphaY = int256.MustFromDec("5535253426143432140214225325")
			tauBetaX  = int256.MustFromDec("21543253245215")
			tauBetaY  = int256.MustFromDec("64215215325345234552542534246431")
			u         = int256.MustFromDec("115363465452543623263253253524")
			v         = int256.MustFromDec("32657394275394645")
			w         = int256.MustFromDec("2693042832140325558430853402583532")
			z         = int256.MustFromDec("644304231583905215")
			dSq       = int256.MustFromDec("900000000000000000000000000")

			// [514215514125231432, 35151251342142152144352142315, 4542145235121424214, 55214231451423345235142324214, 53214232142144365]
			// [[1521421553254332614223423, 5535253426143432140214225325], [21543253245215, 64215215325345234552542534246431], 115363465452543623263253253524, 32657394275394645, 2693042832140325558430853402583532, 644304231583905215, 900000000000000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "35132042025080141711245132546066465504390157966255154622709785937"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("5142158210532504131829479215514125231432")
			beta   = int256.MustFromDec("3515141249871294759214251342142152144352142315")
			c      = int256.MustFromDec("5213489540328503257340254542145235121424214")
			s      = int256.MustFromDec("555124721954935639214231451423345235142324214")
			lambda = int256.MustFromDec("532145234583290581905834902850235830232142144365")

			tauAlphaX = int256.MustFromDec("15214211248957432895732953553254332614223423")
			tauAlphaY = int256.MustFromDec("5535253250812407253405753426143432140214225325")
			tauBetaX  = int256.MustFromDec("2154542357891274975489378759839257843923253245215")
			tauBetaY  = int256.MustFromDec("6452357932534215215325345234552542534246431")
			u         = int256.MustFromDec("11533425937589325763465452543623263253253524")
			v         = int256.MustFromDec("32532976892573895738492719247657394275394645")
			w         = int256.MustFromDec("2693042832140325558430853402583532")
			z         = int256.MustFromDec("644432198753492759325304231583905215")
			dSq       = int256.MustFromDec("900005139214790000000105937458925000000000000000")

			// [5142158210532504131829479215514125231432, 3515141249871294759214251342142152144352142315, 5213489540328503257340254542145235121424214, 555124721954935639214231451423345235142324214, 532145234583290581905834902850235830232142144365]
			// [[15214211248957432895732953553254332614223423, 5535253250812407253405753426143432140214225325], [2154542357891274975489378759839257843923253245215, 6452357932534215215325345234552542534246431], 11533425937589325763465452543623263253253524, 32532976892573895738492719247657394275394645, 2693042832140325558430853402583532, 644432198753492759325304231583905215, 900005139214790000000105937458925000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "-82366328799419662971603887815112727082084762053888506841678"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("6. should return error", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("5142158210532504131829479215514125231432")
			beta   = int256.MustFromDec("3515141249871294759214251342142152144352142315")
			c      = int256.MustFromDec("5213489540328503257340254542145235121424214")
			s      = int256.MustFromDec("555124721954935639214231451423345235142324214")
			lambda = int256.MustFromDec("532145234834902850235830232142144365")

			tauAlphaX = int256.MustFromDec("1521423254332614223423")
			tauAlphaY = int256.MustFromDec("5535214225325")
			tauBetaX  = int256.MustFromDec("215454235789245215")
			tauBetaY  = int256.MustFromDec("6452342534246431")
			u         = int256.MustFromDec("11523263253253524")
			v         = int256.MustFromDec("3253297684275394645")
			w         = int256.MustFromDec("2693042853402583532")
			z         = int256.MustFromDec("64443204231583905215")
			dSq       = int256.MustFromDec("90000537458925000000000000000")

			// [5142158210532504131829479215514125231432, 3515141249871294759214251342142152144352142315, 5213489540328503257340254542145235121424214, 555124721954935639214231451423345235142324214, 532145234834902850235830232142144365]
			// [[1521423254332614223423, 5535214225325], [215454235789245215, 6452342534246431], 11523263253253524, 3253297684275394645, 2693042853402583532, 64443204231583905215, 90000537458925000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected := "56240419042702837047027788699"

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected, actual.Dec())
	})
}

func Test_calcAtAChi(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("2153421421514125231432")
			beta   = int256.MustFromDec("351242142152144352142315")
			c      = int256.MustFromDec("4542142342121424214")
			s      = int256.MustFromDec("552142314423142324214")
			lambda = int256.MustFromDec("532142342142142144365")

			tauAlphaX = int256.MustFromDec("152142154332623423")
			tauAlphaY = int256.MustFromDec("55353432140214225325")
			tauBetaX  = int256.MustFromDec("215432632543253245215")
			tauBetaY  = int256.MustFromDec("6421521532542534246431")
			u         = int256.MustFromDec("1153452543263253253524")
			v         = int256.MustFromDec("326533539257394275394645")
			w         = int256.MustFromDec("269304283058430853402583532")
			z         = int256.MustFromDec("64430286340583402583905215")
			dSq       = int256.MustFromDec("20000000000000000000000000000000000000000")

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}

			x = int256.MustFromDec("2153421421514125231432")
			y = int256.MustFromDec("351242142152144352142315")
		)

		// [2153421421514125231432, 351242142152144352142315, 4542142342121424214, 552142314423142324214, 532142342142142144365]
		// [[152142154332623423, 55353432140214225325], [215432632543253245215, 6421521532542534246431], 1153452543263253253524, 326533539257394275394645, 269304283058430853402583532, 64430286340583402583905215, 20000000000000000000000000000000000000000]

		expected := "-38885251804"
		actual, err := GyroECLPMath.calcAtAChi(x, y, p, d)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("2153443253253221421514125231432")
			beta   = int256.MustFromDec("35124214523562352152144352142315")
			c      = int256.MustFromDec("45421645523553256342342121424214")
			s      = int256.MustFromDec("5526346462535342314423142324214")
			lambda = int256.MustFromDec("53264326325632142342142142144365")

			tauAlphaX = int256.MustFromDec("1535235325264362142154332623423")
			tauAlphaY = int256.MustFromDec("5535325634765453432140214225325")
			tauBetaX  = int256.MustFromDec("236625463415432632543253245215")
			tauBetaY  = int256.MustFromDec("647352355436421521532542534246431")
			u         = int256.MustFromDec("1153523464374352543263253253524")
			v         = int256.MustFromDec("326564536374637539257394275394645")
			w         = int256.MustFromDec("26938453647504283058430853402583532")
			z         = int256.MustFromDec("644302632286523252340583402583905215")
			dSq       = int256.MustFromDec("20000532000000054325083200062350000000000")

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}

			x = int256.MustFromDec("215353254255432532421514125231432")
			y = int256.MustFromDec("3516623436253242142152144352142315")
		)

		// [215353254255432532421514125231432, 3516623436253242142152144352142315, 45421645523553256342342121424214, 5526346462535342314423142324214, 53264326325632142342142142144365]
		// [[1535235325264362142154332623423, 5535325634765453432140214225325], [236625463415432632543253245215, 647352355436421521532542534246431], 1153523464374352543263253253524, 326564536374637539257394275394645, 26938453647504283058430853402583532, 644302632286523252340583402583905215, 20000532000000054325083200062350000000000]

		expected := "-7799585557653041349097132007434611812627031463388"
		actual, err := GyroECLPMath.calcAtAChi(x, y, p, d)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("215344325325322142151441636498205823905125231432")
			beta   = int256.MustFromDec("35124214523562352152532573896795325144352142315")
			c      = int256.MustFromDec("4542164552355325652357239532759342342121424214")
			s      = int256.MustFromDec("55263464625353423144521894712947214823142324214")
			lambda = int256.MustFromDec("5326432632563543972589179512142342142142144365")

			tauAlphaX = int256.MustFromDec("15352362398753295738925325264362142154332623423")
			tauAlphaY = int256.MustFromDec("553532562578239579674953234765453432140214225325")
			tauBetaX  = int256.MustFromDec("23662546341623759327925732955432632543253245215")
			tauBetaY  = int256.MustFromDec("647352355625752935789354436421521532542534246431")
			u         = int256.MustFromDec("115352346256453987219571964374352543263253253524")
			v         = int256.MustFromDec("3265645363746235325353525637539257394275394645")
			w         = int256.MustFromDec("26938453647552404534253283058430853402583532")
			z         = int256.MustFromDec("644302632286532534266523252340583402583905215")
			dSq       = int256.MustFromDec("200005325124000000054325083200062350000000000")

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}

			x = int256.MustFromDec("21535343214254255432532421514125231432")
			y = int256.MustFromDec("3516623543214436253242142152144352142315")
		)

		// [21535343214254255432532421514125231432, 3516623543214436253242142152144352142315, 4542164552355325652357239532759342342121424214, 55263464625353423144521894712947214823142324214, 5326432632563543972589179512142342142142144365]
		// [[15352362398753295738925325264362142154332623423, 553532562578239579674953234765453432140214225325], [23662546341623759327925732955432632543253245215, 647352355625752935789354436421521532542534246431], 115352346256453987219571964374352543263253253524, 3265645363746235325353525637539257394275394645, 26938453647552404534253283058430853402583532, 644302632286532534266523252340583402583905215, 200005325124000000054325083200062350000000000]

		expected := "7887132146191944884544337407379019375416412348150335990912"
		actual, err := GyroECLPMath.calcAtAChi(x, y, p, d)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("215344325325322205823905125231432")
			beta   = int256.MustFromDec("35124214523562352152532573896795325144352142315")
			c      = int256.MustFromDec("4542164552355325652357239532759342342121424214")
			s      = int256.MustFromDec("55263464625353423144521894712947214823142324214")
			lambda = int256.MustFromDec("5326432632563543972589179512142342142142144365")

			tauAlphaX = int256.MustFromDec("15352362398753295738925325264362142154332623423")
			tauAlphaY = int256.MustFromDec("55353256257234765453432140214225325")
			tauBetaX  = int256.MustFromDec("23662546341623759327925732955432632543253245215")
			tauBetaY  = int256.MustFromDec("647352355625752935789354436421521532542534246431")
			u         = int256.MustFromDec("11535234625645319571964374352543263253253524")
			v         = int256.MustFromDec("3265645363746235325353525637539257394275394645")
			w         = int256.MustFromDec("26938453647552404534253283058430853402583532")
			z         = int256.MustFromDec("644302632286532534266523252340583402583905215")
			dSq       = int256.MustFromDec("2000053251240062350000000000")

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}

			x = int256.MustFromDec("21535343421514125231432")
			y = int256.MustFromDec("3516623543214436244352142315")
		)

		// [21535343421514125231432, 3516623543214436244352142315, 4542164552355325652357239532759342342121424214, 55263464625353423144521894712947214823142324214, 5326432632563543972589179512142342142142144365]
		// [[15352362398753295738925325264362142154332623423, 55353256257234765453432140214225325], [23662546341623759327925732955432632543253245215, 647352355625752935789354436421521532542534246431], 11535234625645319571964374352543263253253524, 3265645363746235325353525637539257394275394645, 26938453647552404534253283058430853402583532, 644302632286532534266523252340583402583905215, 2000053251240062350000000000]

		expected := "3988616482385571253294877363629540588280649670179831045462"
		actual, err := GyroECLPMath.calcAtAChi(x, y, p, d)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("215344325325322212452105823905125231432")
			beta   = int256.MustFromDec("35124214523562352152532573896795325144352142315")
			c      = int256.MustFromDec("454216455235532565235751947239532759342342121424214")
			s      = int256.MustFromDec("552634646253534231445521421894712947214823142324214")
			lambda = int256.MustFromDec("5326432632563543972589179512142342142142144365")

			tauAlphaX = int256.MustFromDec("15352362398753295738925325264362142154332623423")
			tauAlphaY = int256.MustFromDec("55353256257234765453432140214225325")
			tauBetaX  = int256.MustFromDec("23662546341623759327925732955432632543253245215")
			tauBetaY  = int256.MustFromDec("647352355625752935789354436421521532542534246431")
			u         = int256.MustFromDec("11535234625645319571964374352543263253253524")
			v         = int256.MustFromDec("512421497591872319571964791654372965721986574392")
			w         = int256.MustFromDec("26938453647552404534253283058430853402583532")
			z         = int256.MustFromDec("6443026322851416532534266523252340583402583905215")
			dSq       = int256.MustFromDec("20000533124214921759251240062350000000000")

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}

			x = int256.MustFromDec("215424892175921435343421514125231432")
			y = int256.MustFromDec("35166512974812974291569223543214436244352142315")
		)

		// [215424892175921435343421514125231432, 35166512974812974291569223543214436244352142315, 454216455235532565235751947239532759342342121424214, 552634646253534231445521421894712947214823142324214, 5326432632563543972589179512142342142142144365]
		// [[15352362398753295738925325264362142154332623423, 55353256257234765453432140214225325], [23662546341623759327925732955432632543253245215, 647352355625752935789354436421521532542534246431], 11535234625645319571964374352543263253253524, 512421497591872319571964791654372965721986574392, 26938453647552404534253283058430853402583532, 6443026322851416532534266523252340583402583905215, 20000533124214921759251240062350000000000]

		expected := "4408952888014525817065143458921540166316832770816371263753"
		actual, err := GyroECLPMath.calcAtAChi(x, y, p, d)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})
}

func Test_virtualOffset0(t *testing.T) {

}

func Test_virtualOffset1(t *testing.T) {

}

func Test_maxBalances0(t *testing.T) {

}

func Test_maxBalances1(t *testing.T) {

}

func Test_calcMinAtxAChiySqPlusAtxSq(t *testing.T) {

}

func Test_calc2AtxAtyAChixAChiy(t *testing.T) {

}

func Test_calcMinAtyAChixSqPlusAtySq(t *testing.T) {

}

func Test_calcInvariantSqrt(t *testing.T) {

}

func Test_checkAssetBounds(t *testing.T) {

}

func Test_calcXpXpDivLambdaLambda(t *testing.T) {

}

func Test_solveQuadraticSwap(t *testing.T) {

}

func Test_calcYGivenX(t *testing.T) {

}

func Test_calcXGivenY(t *testing.T) {

}
